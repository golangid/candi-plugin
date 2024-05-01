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
	"github.com/golangid/candi/codebase/interfaces"
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

	bk       *Broker
	handlers map[string]types.WorkerHandler
}

// NewSTOMPWorker create new stomp client worker for subscribe from queue
func NewSTOMPWorker(service factory.ServiceFactory, broker interfaces.Broker, opts ...OptionFunc) factory.AppServerFactory {
	stompBk, ok := broker.(*Broker)
	if !ok {
		panic("Missing STOMP broker, make sure STOMP has been registered to broker in service config")
	}

	worker := &workerEngine{
		service: service,
		bk:      stompBk,
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
		if h := m.WorkerHandler(worker.bk.WorkerType); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				if _, ok := worker.handlers[handler.Pattern]; ok {
					log.Panicf("STOMP Worker%s: warning, topic %s has been used in another module, overwrite handler func",
						getWorkerTypeLog(worker.bk.WorkerType), handler.Pattern)
				}

				worker.handlers[handler.Pattern] = handler
				sub, err := worker.bk.Conn.Subscribe(handler.Pattern, stomp.AckClient)
				if err != nil {
					log.Panicf("STOMP%s: cannot subscribe to %s: %s", getWorkerTypeLog(worker.bk.WorkerType), handler.Pattern, err.Error())
				}

				logger.LogYellow(fmt.Sprintf("[STOMP-WORKER]%s (topic): %-8s  (consumed by module)--> [%s]",
					getWorkerTypeLog(worker.bk.WorkerType), handler.Pattern, m.Name()))

				worker.channels = append(worker.channels, reflect.SelectCase{
					Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub.C),
				})
				worker.semaphore[handler.Pattern] = make(chan struct{}, 1)
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ STOMP worker%s running with %d topics. Broker: %s\x1b[0m\n\n",
		getWorkerTypeLog(worker.bk.WorkerType), len(worker.handlers), worker.bk.Conn.Server())
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
	log.Printf("\x1b[33;1mStopping STOMP Worker%s...\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	defer func() {
		log.Printf("\x1b[33;1mStopping STOMP Worker%s:\x1b\n \x1b[32;1mSUCCESS\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType))
	}()

	w.shutdown <- struct{}{}
	w.isShutdown = true
	runningJob := 0
	for _, sem := range w.semaphore {
		runningJob += len(sem)
	}
	if runningJob != 0 {
		fmt.Printf("\x1b[34;1mSTOMP Worker%s:\x1b[0m waiting %d job until done...\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType), runningJob)
	}

	w.wg.Wait()
}

func (w *workerEngine) Name() string {
	return string(w.bk.WorkerType)
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

	if w.opt.debugMode {
		log.Printf("\x1b[35;3mSTOMP Worker%s: consuming message from '%s'\x1b[0m\n", getWorkerTypeLog(w.bk.WorkerType), msg.Destination)
	}

	if w.bk.WorkerType != STOMPBroker {
		trace.SetTag("worker_type", string(w.bk.WorkerType))
	}
	trace.SetTag("broker", msg.Conn.Server())
	trace.SetTag("destination", msg.Destination)
	trace.SetTag("content-type", msg.ContentType)
	trace.Log("message.body", msg.Body)

	var eventContext candishared.EventContext
	eventContext.SetContext(ctx)
	eventContext.SetWorkerType(w.Name())
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

func getWorkerTypeLog(name types.Worker) (workerType string) {
	if name != STOMPBroker {
		workerType = " [worker_type: " + string(name) + "]"
	}
	return
}
