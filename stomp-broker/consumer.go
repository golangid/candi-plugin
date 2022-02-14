package stompbroker

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/go-stomp/stomp/v3"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"
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

	conn     *stomp.Conn
	handlers map[string]types.WorkerHandler
}

// NewSTOMPWorker create new stomp client worker for subscribe from queue
func NewSTOMPWorker(service factory.ServiceFactory, opts ...OptionFunc) factory.AppServerFactory {
	if service.GetDependency().GetBroker(STOMPBroker) == nil {
		panic("Missing STOMP configuration, make sure STOMP has been registered to broker in service config")
	}

	conn := service.GetDependency().GetBroker(STOMPBroker).GetConfiguration().(*stomp.Conn)
	worker := &workerEngine{
		service: service,
		conn:    conn,
		opt:     getDefaultOption(),
	}

	for _, opt := range opts {
		opt(&worker.opt)
	}

	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())
	worker.shutdown = make(chan struct{}, 1)
	worker.handlers = make(map[string]types.WorkerHandler)
	worker.semaphore = make(map[string]chan struct{})

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(STOMPBroker); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				if _, ok := worker.handlers[handler.Pattern]; ok {
					panic(fmt.Errorf("STOMP Worker: warning, topic %s has been used in another module, overwrite handler func", handler.Pattern))
				}

				worker.handlers[handler.Pattern] = handler
				sub, err := conn.Subscribe(handler.Pattern, stomp.AckClient)
				if err != nil {
					panic(fmt.Errorf("STOMP: cannot subscribe to %s: %s", handler.Pattern, err.Error()))
				}

				logger.LogYellow(fmt.Sprintf("[STOMP-WORKER] (topic): %-8s  (consumed by module)--> [%s]", handler.Pattern, m.Name()))
				worker.channels = append(worker.channels, reflect.SelectCase{
					Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub.C),
				})
				worker.semaphore[handler.Pattern] = make(chan struct{}, 1)
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ STOMP worker running with %d topics. Broker: %s\x1b[0m\n\n", len(worker.handlers), conn.Server())
	return worker
}

func (w *workerEngine) Serve() {

	for {
		select {
		case <-w.shutdown:
			return

		default:
		}

		chosen, value, ok := reflect.Select(w.channels)
		if !ok {
			continue
		}

		if msg, ok := value.Interface().(*stomp.Message); ok {
			w.semaphore[msg.Destination] <- struct{}{}
			if w.isShutdown {
				return
			}

			w.wg.Add(1)
			go func(idx int, message *stomp.Message) {
				w.processMessage(message)
				w.wg.Done()
				<-w.semaphore[message.Destination]
			}(chosen, msg)
		}
	}
}

func (w *workerEngine) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping STOMP Worker...\x1b[0m")
	defer func() { log.Println("\x1b[33;1mStopping STOMP Worker:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	w.shutdown <- struct{}{}
	w.isShutdown = true
	runningJob := 0
	for _, sem := range w.semaphore {
		runningJob += len(sem)
	}
	if runningJob != 0 {
		fmt.Printf("\x1b[34;1mSTOMP Worker:\x1b[0m waiting %d job until done...\x1b[0m\n", runningJob)
	}

	w.wg.Wait()
}

func (w *workerEngine) Name() string {
	return string(STOMPBroker)
}

func (w *workerEngine) processMessage(msg *stomp.Message) {
	if w.ctx.Err() != nil {
		logger.LogRed(w.Name() + " > ctx root err: " + w.ctx.Err().Error())
		return
	}

	eventID := msg.Header.Get(StompEventID)
	if eventID != "" {
		// lock for multiple worker (if running on multiple pods/instance)
		if w.opt.locker.IsLocked(w.getLockKey(eventID)) {
			return
		}
		defer w.opt.locker.Unlock(w.getLockKey(eventID))
	}

	ctx := w.ctx
	selectedHandler := w.handlers[msg.Destination]
	if selectedHandler.DisableTrace {
		ctx = tracer.SkipTraceContext(ctx)
	}

	header := map[string]string{}
	for i := 0; i < msg.Header.Len(); i++ {
		k, v := msg.Header.GetAt(i)
		header[k] = v
	}

	trace, ctx := tracer.StartTraceFromHeader(w.ctx, "STOMPWorker", header)
	defer func() {
		if r := recover(); r != nil {
			trace.SetError(fmt.Errorf("panic: %v", r))
		}

		if selectedHandler.AutoACK {
			msg.Conn.Ack(msg)
		}
		logger.LogGreen(w.Name() + " > trace_url: " + tracer.GetTraceURL(ctx))
		trace.SetTag("trace_id", tracer.GetTraceID(ctx))
		trace.Finish()
	}()

	log.Printf("\x1b[35;3mSTOMP Worker: consuming message from '%s'\x1b[0m", msg.Destination)

	trace.SetTag("broker", msg.Conn.Server())
	trace.SetTag("destination", msg.Destination)
	trace.SetTag("content-type", msg.ContentType)
	trace.Log("message.body", msg.Body)

	var eventContext candishared.EventContext
	eventContext.SetContext(ctx)
	eventContext.SetWorkerType(string(STOMPBroker))
	eventContext.SetHandlerRoute(msg.Destination)
	eventContext.SetHeader(header)
	eventContext.SetKey(eventID)
	eventContext.Write(msg.Body)

	for _, handlerFunc := range selectedHandler.HandlerFuncs {
		if err := handlerFunc(&eventContext); err != nil {
			eventContext.SetError(err)
			trace.SetError(err)
		}
	}
}

func (w *workerEngine) getLockKey(eventID string) string {
	return fmt.Sprintf("%s:stomp-broker-lock:%s", w.service.Name(), eventID)
}
