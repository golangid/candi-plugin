package gcppubsub

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
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

	semaphore map[string]chan struct{}
	shutdown  chan struct{}
	bk        *Broker

	subscribers map[string]*pubsub.Subscription
	handlers    map[string]types.WorkerHandler
}

// NewPubSubWorker create new gcp pubsub consumer, throw panic if error happened
func NewPubSubWorker(service factory.ServiceFactory, broker interfaces.Broker, subscriberID string) factory.AppServerFactory {
	gcpBk, ok := broker.(*Broker)
	if !ok {
		panic("Missing GCP PubSub broker, make sure GCP PubSub has been registered to broker in service config")
	}

	worker := new(workerEngine)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())

	worker.bk = gcpBk
	worker.shutdown = make(chan struct{})
	worker.subscribers = make(map[string]*pubsub.Subscription)
	worker.handlers = make(map[string]types.WorkerHandler)
	worker.semaphore = make(map[string]chan struct{})

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(gcpBk.WorkerType); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				topic := worker.createTopic(handler.Pattern)
				worker.subscribers[handler.Pattern] = worker.createSubscription(subscriberID+"_"+handler.Pattern, topic)

				logger.LogYellow(fmt.Sprintf(`[GCPPUBSUB-CONSUMER]%s (topic): %-15s  --> (module): "%s"`, getWorkerTypeLog(gcpBk.WorkerType), `"`+handler.Pattern+`"`, m.Name()))
				worker.handlers[handler.Pattern] = handler
				worker.semaphore[handler.Pattern] = make(chan struct{}, 1)
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ GCP PubSub consumer%s running with %d topics. Subscriber ID: %s\x1b[0m\n\n", getWorkerTypeLog(gcpBk.WorkerType),
		len(worker.handlers), subscriberID)

	return worker
}

func (w *workerEngine) Serve() {
	for topic, subs := range w.subscribers {
		go func(sub *pubsub.Subscription, topic string) {
			sub.Receive(w.ctx, func(ctx context.Context, msg *pubsub.Message) { w.processMessage(ctx, topic, msg) })
		}(subs, topic)
	}

	<-w.shutdown
}

func (w *workerEngine) Shutdown(ctx context.Context) {
	log.Printf("\x1b[33;1mStopping GCP PubSub Worker%s...\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	defer func() {
		recover()
		log.Printf("\x1b[33;1mStopping GCP PubSub Worker%s:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	}()

	w.ctxCancelFunc()
	w.shutdown <- struct{}{}
	w.bk.Client.Close()
}

func (w *workerEngine) Name() string {
	return string(w.bk.WorkerType)
}

func (w *workerEngine) createTopic(topicName string) *pubsub.Topic {
	topic := w.bk.Client.Topic(topicName)
	ok, err := topic.Exists(w.ctx)
	if err != nil {
		panic("GCP PubSub check topic " + topicName + ": " + err.Error())
	}
	if !ok {
		topic, err = w.bk.Client.CreateTopic(w.ctx, topicName)
		if err != nil {
			panic("GCP PubSub create topic " + topicName + ": " + err.Error())
		}
	}
	return topic
}

func (w *workerEngine) createSubscription(subscriberID string, topic *pubsub.Topic) *pubsub.Subscription {
	sub := w.bk.Client.Subscription(subscriberID)
	ok, err := sub.Exists(w.ctx)
	if err != nil {
		panic("GCP PubSub check subscriber " + subscriberID + ": " + err.Error())
	}
	// sub.Update(w.ctx, pubsub.SubscriptionConfigToUpdate{})
	if !ok {
		sub, err = w.bk.Client.CreateSubscription(w.ctx, subscriberID, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 20 * time.Second,
		})
		if err != nil {
			panic("GCP PubSub create subscriber " + subscriberID + ": " + err.Error())
		}
	}
	return sub
}

func (w *workerEngine) processMessage(ctx context.Context, topic string, msg *pubsub.Message) {
	if w.ctx.Err() != nil {
		logger.LogRed("gcppubsub_consumer > ctx root err: " + w.ctx.Err().Error())
		return
	}

	w.semaphore[topic] <- struct{}{}

	go func(topic string, msg *pubsub.Message) {
		defer func() { <-w.semaphore[topic] }()

		selectedHandler := w.handlers[topic]
		if selectedHandler.DisableTrace {
			ctx = tracer.SkipTraceContext(ctx)
		}

		trace, ctx := tracer.StartTraceFromHeader(ctx, "GCPPubSubConsumer", msg.Attributes)
		defer trace.Finish(
			tracer.FinishWithRecoverPanic(func(any) {}),
			tracer.FinishWithFunc(func() {
				if selectedHandler.AutoACK {
					msg.Ack()
				}
			}),
		)

		trace.SetTag("topic", topic)
		trace.Log("attributes", msg.Attributes)
		trace.Log("body", msg.Data)

		log.Printf("\x1b[35;3mGCP PubSub Worker%s: consuming message from topic '%s'\x1b[0m", getWorkerTypeLog(w.bk.WorkerType), topic)

		eventContext := candishared.NewEventContext(bytes.NewBuffer(make([]byte, 0, 256)))
		eventContext.SetContext(ctx)
		eventContext.SetWorkerType(string(w.bk.WorkerType))
		eventContext.SetHandlerRoute(topic)
		eventContext.SetHeader(msg.Attributes)
		eventContext.SetKey(msg.ID)
		eventContext.Write(msg.Data)

		for _, handlerFunc := range selectedHandler.HandlerFuncs {
			if err := handlerFunc(eventContext); err != nil {
				eventContext.SetError(err)
			}
		}
	}(topic, msg)
}

func getWorkerTypeLog(name types.Worker) (workerType string) {
	if name != GoogleCloudPubSub {
		workerType = " [worker_type: " + string(name) + "]"
	}
	return
}
