package amazonsqs

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"
)

type workerEngine struct {
	ctx           context.Context
	ctxCancelFunc func()

	service   factory.ServiceFactory
	semaphore map[string]chan struct{}
	shutdown  chan struct{}
	wg        sync.WaitGroup
	opt       option

	bk               *Broker
	maxNumberMessage int32
	waitTimeSeconds  int32
	queueURLs        map[string]string
	handlers         map[string]types.WorkerHandler
}

// NewAmazonSQSWorker create new amazonsqs client worker for subscribe from queue.
// maxNumberMessage is the max messages fetched per poll (1-10), waitTimeSeconds
// enables long polling (0-20).
func NewAmazonSQSWorker(service factory.ServiceFactory, broker interfaces.Broker, maxNumberMessage int, waitTimeSeconds int, opts ...OptionFunc) factory.AppServerFactory {
	amazonSQSBk, ok := broker.(*Broker)
	if !ok {
		panic("Missing Amazon SQS broker, make sure Amazon SQS has been registered to broker in service config")
	}

	worker := new(workerEngine)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())
	worker.service = service
	worker.bk = amazonSQSBk
	worker.shutdown = make(chan struct{})
	worker.handlers = make(map[string]types.WorkerHandler)
	worker.queueURLs = make(map[string]string)
	worker.semaphore = make(map[string]chan struct{})

	worker.maxNumberMessage = int32(maxNumberMessage)
	if worker.maxNumberMessage <= 0 || worker.maxNumberMessage > 10 {
		worker.maxNumberMessage = 10
	}
	worker.waitTimeSeconds = int32(waitTimeSeconds)
	if worker.waitTimeSeconds < 0 || worker.waitTimeSeconds > 20 {
		worker.waitTimeSeconds = 20
	}

	worker.opt = getDefaultOption()
	for _, opt := range opts {
		opt(&worker.opt)
	}

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(amazonSQSBk.WorkerType); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				queueURL, err := amazonSQSBk.resolveQueueURL(worker.ctx, handler.Pattern)
				if err != nil {
					log.Panicf("AmazonSQS%s: cannot subscribe to %q: %s", getWorkerTypeLog(amazonSQSBk.WorkerType), handler.Pattern, err.Error())
				}
				worker.queueURLs[handler.Pattern] = queueURL
				worker.handlers[handler.Pattern] = handler
				worker.semaphore[handler.Pattern] = make(chan struct{}, worker.maxNumberMessage)

				logger.LogYellow(fmt.Sprintf(`[AmazonSQS-CONSUMER]%s (queue): %-15s --> (module): "%s"`, getWorkerTypeLog(amazonSQSBk.WorkerType), `"`+handler.Pattern+`"`, m.Name()))
			}
		}
	}

	return worker
}

func (w *workerEngine) Serve() {
	if len(w.handlers) == 0 {
		logger.LogYellow("amazonsqs: no worker handler registered, worker is idle")
		<-w.shutdown
		return
	}

	for queueName := range w.handlers {
		w.wg.Add(1)
		go func(queueName string) {
			defer w.wg.Done()
			w.consumeLoop(queueName)
		}(queueName)
	}

	<-w.shutdown
}

// consumeLoop continuously long-polls a single queue until the worker is stopped.
func (w *workerEngine) consumeLoop(queueName string) {
	queueURL := w.queueURLs[queueName]
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		output, err := w.bk.Client.ReceiveMessage(w.ctx, &sqs.ReceiveMessageInput{
			QueueUrl:              aws.String(queueURL),
			MaxNumberOfMessages:   w.maxNumberMessage,
			WaitTimeSeconds:       w.waitTimeSeconds,
			MessageAttributeNames: []string{"All"},
		})
		if err != nil {
			if w.ctx.Err() != nil {
				return
			}
			logger.LogRed("amazonsqs_consumer > receive message from queue " + queueName + ": " + err.Error())
			continue
		}

		for i := range output.Messages {
			w.processMessage(queueName, queueURL, output.Messages[i])
		}
	}
}

func (w *workerEngine) processMessage(queueName, queueURL string, message sqstypes.Message) {
	selectedHandler := w.handlers[queueName]

	w.semaphore[queueName] <- struct{}{}
	w.wg.Add(1)
	go func() {
		defer func() {
			<-w.semaphore[queueName]
			w.wg.Done()
		}()

		ctx := w.ctx
		if selectedHandler.DisableTrace {
			ctx = tracer.SkipTraceContext(ctx)
		}

		header := headerFromMessageAttributes(message.MessageAttributes)
		trace, ctx := tracer.StartTraceFromHeader(ctx, "AmazonSQSConsumer", header)
		defer trace.Finish(tracer.FinishWithRecoverPanic(func(any) {}))

		if w.bk.WorkerType != AmazonSQSBroker {
			trace.SetTag("worker_type", string(w.bk.WorkerType))
		}
		trace.SetTag("topic", queueName)
		trace.SetTag("queue_url", queueURL)
		if message.MessageId != nil {
			trace.SetTag("message_id", *message.MessageId)
		}
		trace.Log("body", message.Body)

		eventContext := candishared.NewEventContext(bytes.NewBuffer(make([]byte, 0, 256)))
		eventContext.SetContext(ctx)
		eventContext.SetWorkerType(string(w.bk.WorkerType))
		eventContext.SetHandlerRoute(queueName)
		eventContext.SetHeader(header)
		if message.MessageId != nil {
			eventContext.SetKey(*message.MessageId)
		}
		if message.Body != nil {
			eventContext.WriteString(*message.Body)
		}

		var handlerErr error
		for _, handlerFunc := range selectedHandler.HandlerFuncs {
			if err := handlerFunc(eventContext); err != nil {
				eventContext.SetError(err)
				handlerErr = err
			}
		}
		trace.SetError(handlerErr)

		// only acknowledge (delete) the message when handled successfully, so a
		// failed message becomes visible again after the queue visibility timeout
		if selectedHandler.AutoACK && handlerErr == nil {
			// use a detached context so the delete still completes during shutdown
			if _, err := w.bk.Client.DeleteMessage(context.WithoutCancel(ctx), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: message.ReceiptHandle,
			}); err != nil {
				logger.LogRed("amazonsqs_consumer > delete message from queue " + queueName + ": " + err.Error())
			}
		}
	}()
}

func (w *workerEngine) Shutdown(ctx context.Context) {
	log.Printf("\x1b[33;1mStopping AmazonSQS Worker%s...\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	defer log.Printf("\x1b[33;1mStopping AmazonSQS Worker%s:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))

	// stop polling loops and unblock Serve
	w.ctxCancelFunc()
	close(w.shutdown)

	runningJob := 0
	for _, sem := range w.semaphore {
		runningJob += len(sem)
	}
	if runningJob != 0 {
		fmt.Printf("\x1b[34;1mAmazonSQS Worker%s:\x1b[0m waiting %d job until done...\n", getWorkerTypeLog(w.bk.WorkerType), runningJob)
	}

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		logger.LogRed("amazonsqs: force shutdown, some in-flight jobs may not be finished")
	}
}

func (w *workerEngine) Name() string {
	return string(w.bk.WorkerType)
}

func getWorkerTypeLog(name types.Worker) (workerType string) {
	if name != AmazonSQSBroker {
		workerType = " [worker_type: " + string(name) + "]"
	}
	return
}
