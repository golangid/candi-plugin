package stompworker

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/go-stomp/stomp/v3"
	"pkg.agungdp.dev/candi/codebase/factory"
	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/logger"
	"pkg.agungdp.dev/candi/tracer"
)

type handlerType struct {
	handlerFunc   types.WorkerHandlerFunc
	errorHandlers []types.WorkerErrorHandler
}

type workerEngine struct {
	ctx           context.Context
	ctxCancelFunc func()

	channels   []reflect.SelectCase
	semaphore  map[string]chan struct{}
	shutdown   chan struct{}
	isShutdown bool
	wg         sync.WaitGroup

	conn     *stomp.Conn
	handlers map[string]handlerType
}

// NewStompWorker create new stomp client worker for subscribe from queue
func NewStompWorker(service factory.ServiceFactory, conn *stomp.Conn) factory.AppServerFactory {
	worker := &workerEngine{
		conn: conn,
	}

	worker.ctx, worker.ctxCancelFunc = context.WithCancel(context.Background())
	worker.shutdown = make(chan struct{}, 1)
	worker.handlers = make(map[string]handlerType)
	worker.semaphore = make(map[string]chan struct{})

	for _, m := range service.GetModules() {
		if h := m.WorkerHandler(STOMPConsumer); h != nil {
			var handlerGroup types.WorkerHandlerGroup
			h.MountHandlers(&handlerGroup)
			for _, handler := range handlerGroup.Handlers {
				if _, ok := worker.handlers[handler.Pattern]; ok {
					panic(fmt.Errorf("STOMP Worker: warning, topic %s has been used in another module, overwrite handler func", handler.Pattern))
				}

				worker.handlers[handler.Pattern] = handlerType{
					handlerFunc: handler.HandlerFunc, errorHandlers: handler.ErrorHandler,
				}

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

	fmt.Printf("\x1b[34;1m⇨ STOMP worker running. Broker: %s\x1b[0m\n\n", conn.Server())
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
	return string(STOMPConsumer)
}

func (w *workerEngine) processMessage(msg *stomp.Message) {
	if w.ctx.Err() != nil {
		logger.LogRed(w.Name() + " > ctx root err: " + w.ctx.Err().Error())
		return
	}

	trace, ctx := tracer.StartTraceWithContext(w.ctx, "STOMPWorker")
	defer func() {
		if r := recover(); r != nil {
			trace.SetError(fmt.Errorf("%v", r))
		}
		msg.Conn.Ack(msg)
		logger.LogGreen(w.Name() + " > trace_url: " + tracer.GetTraceURL(ctx))
		trace.Finish()
	}()

	log.Printf("\x1b[35;3mSTOMP Worker: consuming message from '%s'\x1b[0m", msg.Destination)

	trace.SetTag("broker", msg.Conn.Server())
	trace.SetTag("destination", msg.Destination)
	trace.SetTag("content-type", msg.ContentType)
	tracer.Log(ctx, "message.body", string(msg.Body))

	selectedHandler := w.handlers[msg.Destination]
	if err := selectedHandler.handlerFunc(ctx, msg.Body); err != nil {
		for _, errHandler := range selectedHandler.errorHandlers {
			errHandler(ctx, STOMPConsumer, msg.Destination, msg.Body, err)
		}
		trace.SetError(err)
	}
}
