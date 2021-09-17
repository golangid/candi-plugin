package gcppubsub

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"
)

type workerEngine struct {
	ctx           context.Context
	ctxCancelFunc func()

	semaphore map[string]chan struct{}
	shutdown  chan struct{}
	client    *pubsub.Client

	subscribers map[string]*pubsub.Subscription
	handlers    map[string]types.WorkerHandler
}

// NewPubSubWorker create new gcp pubsub consumer, throw panic if error happened
func NewPubSubWorker(service factory.ServiceFactory, subscriberID string) factory.AppServerFactory {
	if service.GetDependency().GetBroker(GoogleCloudPubSub) == nil {
		panic("Missing GCP PubSub configuration, make sure GCP PubSub has been registered to broker in service config")
	}

	worker := new(workerEngine)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())

	worker.client = service.GetDependency().GetBroker(GoogleCloudPubSub).GetConfiguration().(*pubsub.Client)
	worker.shutdown = make(chan struct{})
	worker.subscribers = make(map[string]*pubsub.Subscription)
	worker.handlers = make(map[string]types.WorkerHandler)
	worker.semaphore = make(map[string]chan struct{})

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(GoogleCloudPubSub); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				topic := worker.createTopic(handler.Pattern)
				worker.subscribers[handler.Pattern] = worker.createSubscription(subscriberID+"_"+handler.Pattern, topic)

				logger.LogYellow(fmt.Sprintf(`[GCPPUBSUB-CONSUMER] (topic): %-15s  --> (module): "%s"`, `"`+handler.Pattern+`"`, m.Name()))
				worker.handlers[handler.Pattern] = handler
				worker.semaphore[handler.Pattern] = make(chan struct{}, 1)
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ GCP PubSub consumer running with %d topics. Subscriber ID: %s\x1b[0m\n\n",
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
	log.Println("\x1b[33;1mStopping GCP PubSub Worker...\x1b[0m")
	defer func() {
		recover()
		log.Println("\x1b[33;1mStopping GCP PubSub Worker:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m")
	}()

	w.ctxCancelFunc()
	w.shutdown <- struct{}{}
	w.client.Close()
}

func (w *workerEngine) Name() string {
	return string(GoogleCloudPubSub)
}

func (w *workerEngine) createTopic(topicName string) *pubsub.Topic {
	topic := w.client.Topic(topicName)
	ok, err := topic.Exists(w.ctx)
	if err != nil {
		panic("GCP PubSub: " + err.Error())
	}
	if !ok {
		topic, err = w.client.CreateTopic(w.ctx, topicName)
		if err != nil {
			panic("GCP PubSub: " + err.Error())
		}
	}
	return topic
}

func (w *workerEngine) createSubscription(subscriberID string, topic *pubsub.Topic) *pubsub.Subscription {
	sub := w.client.Subscription(subscriberID)
	ok, err := sub.Exists(w.ctx)
	if err != nil {
		panic("GCP PubSub: " + err.Error())
	}
	if !ok {
		sub, err = w.client.CreateSubscription(w.ctx, subscriberID, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 20 * time.Second,
		})
		if err != nil {
			panic("GCP PubSub: " + err.Error())
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

		var err error
		trace, ctx := tracer.StartTraceWithContext(ctx, "GCPPubSubConsumer")
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}

			if selectedHandler.AutoACK {
				msg.Ack()
			}
			trace.SetError(err)
			logger.LogGreen("gcppubsub_consumer > trace_url: " + tracer.GetTraceURL(ctx))
			trace.Finish()
		}()

		trace.SetTag("topic", topic)
		trace.Log("attributes", msg.Attributes)
		trace.Log("body", msg.Data)

		log.Printf("\x1b[35;3mGCP PubSub Worker: consuming message from topic '%s'\x1b[0m", topic)

		ctx = candishared.SetToContext(ctx, MessageAttribute, msg.Attributes)
		if err = selectedHandler.HandlerFunc(ctx, msg.Data); err != nil {
			if selectedHandler.ErrorHandler != nil {
				selectedHandler.ErrorHandler(ctx, GoogleCloudPubSub, topic, msg.Data, err)
			}
		}
	}(topic, msg)
}
