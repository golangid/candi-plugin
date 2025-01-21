package mqttbroker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golangid/candi/candihelper"
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
	semaphore     map[string]chan struct{}
	wg            sync.WaitGroup
	shutdown      chan struct{}

	handlers []handlerConfig
	broker   *Broker
	opt      option
	router   *router
}

type handlerConfig struct {
	topic string
	qos   byte
}

// NewMQTTSubscriber construct mqtt consumer
func NewMQTTSubscriber(service factory.ServiceFactory, broker interfaces.Broker, opts ...OptionFunc) factory.AppServerFactory {
	mqttBroker, ok := broker.(*Broker)
	if !ok {
		panic("Missing MQTT broker, make sure MQTT has been registered to broker in service config")
	}

	worker := new(workerEngine)
	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())
	worker.broker = mqttBroker
	worker.semaphore = make(map[string]chan struct{})
	worker.shutdown = make(chan struct{})
	worker.opt = getDefaultOption()
	worker.router = &router{root: newRouteNode()}
	for _, opt := range opts {
		opt(&worker.opt)
	}

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(mqttBroker.WorkerType); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				logger.LogYellow(fmt.Sprintf(`[MQTT-SUBSCRIBER]%s (topic): %-15s  --> (module): "%s"`, getWorkerTypeLog(mqttBroker.WorkerType), `"`+handler.Pattern+`"`, m.Name()))

				qos, ok := handler.Configs[ConfigHeaderQOS].(byte)
				if !ok {
					qos = mqttBroker.subscriberQOS
				}
				worker.handlers = append(worker.handlers, handlerConfig{
					topic: worker.router.addRoute(handler.Pattern, handler),
					qos:   qos,
				})
				worker.semaphore[handler.Pattern] = make(chan struct{}, 1)
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ MQTT subscriber%s running with %d topics.\x1b[0m\n\n", getWorkerTypeLog(mqttBroker.WorkerType), len(worker.handlers))
	return worker
}

func (w *workerEngine) Serve() {
	for _, handler := range w.handlers {
		w.broker.client.Subscribe(handler.topic, handler.qos, w.processMessage)
	}
	<-w.shutdown
}

func (w *workerEngine) Shutdown(ctx context.Context) {
	defer func() {
		fmt.Printf("\r%s \x1b[33;1mStopping MQTT Subscriber%s:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m%s\n",
			time.Now().Format(candihelper.TimeFormatLogger), getWorkerTypeLog(w.broker.WorkerType), strings.Repeat(" ", 20))
	}()
	fmt.Printf("\r%s \x1b[33;1mStopping MQTT Subscriber%s:\x1b[0m ... ", time.Now().Format(candihelper.TimeFormatLogger), getWorkerTypeLog(w.broker.WorkerType))

	w.wg.Wait()
	w.ctxCancelFunc()
	w.shutdown <- struct{}{}
	w.broker.client.Disconnect(500)
}

func (w *workerEngine) Name() string {
	return string(w.broker.WorkerType)
}

func (w *workerEngine) onConnect(c mqtt.Client) {
	for _, handler := range w.handlers {
		c.Subscribe(handler.topic, handler.qos, w.processMessage)
	}
}

func (w *workerEngine) processMessage(_ mqtt.Client, m mqtt.Message) {
	if w.ctx.Err() != nil {
		logger.LogRed("mqtt_subscriber > ctx root err: " + w.ctx.Err().Error())
		return
	}

	ctx := w.ctx
	selectedHandler, params := w.router.match(m.Topic())
	if selectedHandler.DisableTrace {
		ctx = tracer.SkipTraceContext(ctx)
	}
	w.wg.Add(1)
	var err error
	trace, ctx := tracer.StartTraceFromHeader(ctx, "MQTTSubscriber", map[string]string{})
	defer func() {
		if r := recover(); r != nil {
			trace.SetTag("panic", true)
			err = fmt.Errorf("%v", r)
		}
		if selectedHandler.AutoACK {
			m.Ack()
		}
		trace.Finish(tracer.FinishWithError(err))
		w.wg.Done()
	}()

	if w.broker.WorkerType != MQTTBroker {
		trace.SetTag("worker_type", string(w.broker.WorkerType))
	}
	trace.SetTag("topic", m.Topic())
	trace.Log("params", params)
	trace.Log("body", m.Payload())

	if w.opt.debugMode {
		log.Printf("\x1b[35;3mMQTT Subscriber%s: consuming message from topic '%s'\x1b[0m", getWorkerTypeLog(w.broker.WorkerType), m.Topic())
	}

	eventContext := candishared.NewEventContext(bytes.NewBuffer(make([]byte, 0, 256)))
	eventContext.SetContext(ctx)
	eventContext.SetWorkerType(string(w.broker.WorkerType))
	eventContext.SetHandlerRoute(m.Topic())
	eventContext.SetKey(strconv.Itoa(int(m.MessageID())))
	eventContext.Write(m.Payload())
	eventContext.SetHeader(params) // todo move to context value

	for _, handlerFunc := range selectedHandler.HandlerFuncs {
		if err = handlerFunc(eventContext); err != nil {
			eventContext.SetError(err)
		}
	}
}

func getWorkerTypeLog(name types.Worker) (workerType string) {
	if name != MQTTBroker {
		workerType = " [worker_type: " + string(name) + "]"
	}
	return
}
