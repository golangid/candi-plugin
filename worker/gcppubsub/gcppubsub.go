package gcppubsub

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	"pkg.agungdp.dev/candi/codebase/factory"
	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/logger"
	"pkg.agungdp.dev/candi/tracer"
)

type workerEngine struct {
	ctx           context.Context
	ctxCancelFunc func()

	shutdown chan struct{}
	client   *pubsub.Client

	subscribers map[string]*pubsub.Subscription
	handlers    map[string]types.WorkerHandlerFunc
}

// NewPubSubWorker create new gcp pubsub consumer, throw panic if error happened
// Please set env GOOGLE_APPLICATION_CREDENTIALS=[path to your gcp credentials] for implisit load gcp credential (https://cloud.google.com/docs/authentication/production)
func NewPubSubWorker(service factory.ServiceFactory, gcpProjectName, subscriberID string) factory.AppServerFactory {
	worker := new(workerEngine)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())

	var err error
	worker.client, err = pubsub.NewClient(worker.ctx, gcpProjectName)
	if err != nil {
		panic(err)
	}

	worker.shutdown = make(chan struct{})
	worker.subscribers = make(map[string]*pubsub.Subscription)
	worker.handlers = make(map[string]types.WorkerHandlerFunc)

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(GoogleCloudPubSub); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				topic := worker.createTopic(handler.Pattern)
				worker.subscribers[handler.Pattern] = worker.createSubscription(subscriberID+handler.Pattern, topic)

				logger.LogYellow(fmt.Sprintf(`[GCPPUBSUB-CONSUMER] (topic): %-15s  --> (module): "%s"`, `"`+handler.Pattern+`"`, m.Name()))
				worker.handlers[handler.Pattern] = handler.HandlerFunc
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ GCP PubSub consumer running with %d topics. Project ID: %s\x1b[0m\n\n",
		len(worker.handlers), gcpProjectName)

	return worker
}

func (w *workerEngine) Serve() {
	ch := make(chan struct {
		message *pubsub.Message
		topic   string
	})
	defer close(ch)

	for topic, subs := range w.subscribers {
		go func(sub *pubsub.Subscription, topic string) {
			sub.Receive(w.ctx, func(ctx context.Context, msg *pubsub.Message) {
				ch <- struct {
					message *pubsub.Message
					topic   string
				}{
					message: msg, topic: topic,
				}
			})
		}(subs, topic)
	}

	for {
		select {
		case msg := <-ch:
			w.processMessage(msg.topic, msg.message)
		case <-w.shutdown:
			return
		}
	}
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
		panic(err)
	}
	if !ok {
		topic, err = w.client.CreateTopic(w.ctx, topicName)
		if err != nil {
			panic(err)
		}
	}
	return topic
}

func (w *workerEngine) createSubscription(subscriberID string, topic *pubsub.Topic) *pubsub.Subscription {
	sub := w.client.Subscription(subscriberID)
	ok, err := sub.Exists(w.ctx)
	if err != nil {
		panic(err)
	}
	if !ok {
		sub, err = w.client.CreateSubscription(w.ctx, subscriberID, pubsub.SubscriptionConfig{
			Topic:       topic,
			AckDeadline: 20 * time.Second,
		})
		if err != nil {
			panic(err)
		}
	}
	return sub
}

func (w *workerEngine) processMessage(topic string, msg *pubsub.Message) {
	if w.ctx.Err() != nil {
		logger.LogRed("gcppubsub_consumer > ctx root err: " + w.ctx.Err().Error())
		return
	}

	var err error
	trace, ctx := tracer.StartTraceWithContext(w.ctx, "GCPPubSubConsumer")
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
		msg.Ack()
		trace.SetError(err)
		logger.LogGreen("gcppubsub_consumer > trace_url: " + tracer.GetTraceURL(ctx))
		trace.Finish()
	}()

	trace.SetTag("topic", topic)
	trace.Log("attributes", msg.Attributes)
	trace.Log("body", msg.Data)
	err = w.handlers[topic](ctx, msg.Data)
}
