package amazonsqs

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"
	"log"
	"reflect"
	"sync"
)

type workerEngine struct {
	ctx           context.Context
	ctxCancelFunc func()

	service    factory.ServiceFactory
	channels   []reflect.SelectCase
	semaphore  map[string]chan struct{}
	shutdown   chan struct{}
	isShutdown bool
	wg         sync.WaitGroup
	opt        option

	bk          *Broker
	subscribers map[string]*sqs.ReceiveMessageOutput
	queueUrl    map[string]*sqs.GetQueueUrlOutput
	handlers    map[string]types.WorkerHandler
}

// NewAmazonSQSWorker create new amazonsqs client worker for subscribe from queue
func NewAmazonSQSWorker(service factory.ServiceFactory, broker interfaces.Broker, maxNumberMessage int, waitTimeSeconds int) factory.AppServerFactory {
	amazonSQSBk, ok := broker.(*Broker)
	if !ok {
		panic("Missing Amazon SQS broker, make sure Amazon SQS has been registered to broker in service config")
	}

	worker := new(workerEngine)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())

	worker.bk = amazonSQSBk
	worker.shutdown = make(chan struct{})
	worker.handlers = make(map[string]types.WorkerHandler)
	worker.semaphore = make(map[string]chan struct{})

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(amazonSQSBk.WorkerType); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				queueUrlResult, err := worker.bk.Client.GetQueueUrl(worker.ctx, &sqs.GetQueueUrlInput{
					QueueName: aws.String(handler.Pattern),
				})
				if err != nil {
					log.Panicf("AmazonSQS%s: cannot subscribe to %s: %s", getWorkerTypeLog(worker.bk.WorkerType), handler.Pattern, err.Error())
				}
				result, err := worker.bk.Client.ReceiveMessage(worker.ctx, &sqs.ReceiveMessageInput{
					QueueUrl:            queueUrlResult.QueueUrl,
					MaxNumberOfMessages: int32(maxNumberMessage),
					WaitTimeSeconds:     int32(waitTimeSeconds),
				})
				if err != nil {
					log.Panicf("AmazonSQS%s: cannot subscribe to %s: %s", getWorkerTypeLog(worker.bk.WorkerType), handler.Pattern, err.Error())
				}
				worker.subscribers[handler.Pattern] = result
				worker.queueUrl[handler.Pattern] = queueUrlResult

				logger.LogYellow(fmt.Sprintf(`[AmazonSQS-CONSUMER]%s (queue): %-15s  --> (module): "%s"`, getWorkerTypeLog(amazonSQSBk.WorkerType), `"`+handler.Pattern+`"`, m.Name()))
				worker.handlers[handler.Pattern] = handler
				worker.semaphore[handler.Pattern] = make(chan struct{}, 1)
			}
		}
	}

	return worker
}

func (w *workerEngine) Serve() {
	for queue, s := range w.subscribers {
		go func(queue string, s *sqs.ReceiveMessageOutput) {
			w.processMessage(w.ctx, queue, s)
		}(queue, s)
	}

	<-w.shutdown
}

func (w *workerEngine) Shutdown(ctx context.Context) {
	log.Printf("\x1b[33;1mStopping AmazonSQS Worker%s...\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	defer func() {
		log.Printf("\x1b[33;1mStopping AmazonSQS Worker%s:\x1b\n \x1b[32;1mSUCCESS\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	}()

	w.shutdown <- struct{}{}
	w.isShutdown = true
	runningJob := 0
	for _, sem := range w.semaphore {
		runningJob += len(sem)
	}
	if runningJob != 0 {
		fmt.Printf("\x1b[34;1mAmazonSQS Worker%s:\x1b[0m waiting %d job until done...\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType), runningJob)
	}

	w.wg.Wait()
}

func (w *workerEngine) Name() string {
	return string(w.bk.WorkerType)
}

func (w *workerEngine) processMessage(ctx context.Context, queue string, msg *sqs.ReceiveMessageOutput) {
	if w.ctx.Err() != nil {
		logger.LogRed("amazonsqs_consumer > ctx root err: " + w.ctx.Err().Error())
		return
	}

	w.semaphore[queue] <- struct{}{}

	go func(topic string, msg *sqs.ReceiveMessageOutput) {
		defer func() { <-w.semaphore[topic] }()

		selectedHandler := w.handlers[topic]
		if selectedHandler.DisableTrace {
			ctx = tracer.SkipTraceContext(ctx)
		}

		for _, m := range msg.Messages {
			trace, ctx := tracer.StartTraceFromHeader(ctx, "AmazonSQSConsumer", m.Attributes)

			if w.bk.WorkerType != AmazonSQSBroker {
				trace.SetTag("worker_type", string(w.bk.WorkerType))
			}

			trace.SetTag("topic", topic)
			trace.Log("attributes", msg.ResultMetadata)

			trace.Log("message_id", m.MessageId)
			trace.Log("body", m.Body)
			eventContext := candishared.NewEventContext(bytes.NewBuffer(make([]byte, 0, 256)))
			eventContext.SetContext(ctx)
			eventContext.SetWorkerType(string(w.bk.WorkerType))
			eventContext.SetHandlerRoute(topic)
			eventContext.SetHeader(m.Attributes)
			if m.MessageId != nil {
				eventContext.SetKey(*m.MessageId)
			}
			if m.Body != nil {
				eventContext.Write([]byte(*m.Body))
			}

			for _, handlerFunc := range selectedHandler.HandlerFuncs {
				if err := handlerFunc(eventContext); err != nil {
					eventContext.SetError(err)
				}
			}

			trace.Finish(
				tracer.FinishWithRecoverPanic(func(any) {}),
				tracer.FinishWithFunc(func() {
					if selectedHandler.AutoACK {
						selectedQueueUrl := w.queueUrl[topic]
						w.bk.Client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
							QueueUrl:      selectedQueueUrl.QueueUrl,
							ReceiptHandle: m.ReceiptHandle,
						})
					}
				}),
			)
		}

		log.Printf("\x1b[35;3mAmazonSQS Worker%s: consuming message from topic '%s'\x1b[0m", getWorkerTypeLog(w.bk.WorkerType), topic)
	}(queue, msg)
}

func (w *workerEngine) getLockKey(eventID string) string {
	return fmt.Sprintf("%s:amazonsqs-broker-lock:%s", w.service.Name(), eventID)
}

func getWorkerTypeLog(name types.Worker) (workerType string) {
	if name != AmazonSQSBroker {
		workerType = " [worker_type: " + string(name) + "]"
	}
	return
}
