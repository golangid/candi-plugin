package p2p

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"pkg.agungdp.dev/candi/codebase/factory"
	"pkg.agungdp.dev/candi/logger"
	"pkg.agungdp.dev/candi/tracer"
)

type p2pTCP struct {
	option

	listener  net.Listener
	handlers  map[string]HandlerFunc
	semaphore chan struct{}
}

// NewP2PTCP init p2p in TCP network
func NewP2PTCP(service factory.ServiceFactory, addr string, opts ...ServerOption) factory.AppServerFactory {
	srv := new(p2pTCP)
	srv.bufferSize = 1024
	srv.maxConcurrentClient = 100
	srv.enableTracing = true

	for _, opt := range opts {
		opt(&srv.option)
	}

	srv.semaphore = make(chan struct{}, srv.maxConcurrentClient)
	srv.handlers = make(map[string]HandlerFunc)

	var err error
	srv.listener, err = net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	for _, m := range service.GetModules() {
		if h := m.ServerHandler(P2PTCP); h != nil {
			var handlers HandlerGroup
			h.MountHandlers(&handlers)
			for _, handler := range handlers.Handlers {
				srv.handlers[handler.Prefix] = handler.HandlerFunc
				logger.LogYellow(fmt.Sprintf(`[P2P_TCP] (route): %-15s  --> (module): "%s"`, `"`+handler.Prefix+`"`, m.Name()))
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ P2P TCP running with %d route and port [::]%s\x1b[0m\n\n", len(srv.handlers), addr)
	return srv
}

func (s *p2pTCP) Serve() {
	buffer := make([]byte, s.bufferSize)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}

		s.semaphore <- struct{}{}
		go func(conn net.Conn, buff []byte) {
			defer func() { conn.Close(); <-s.semaphore }()

			messages := bytes.Split(bytes.TrimSpace(buff), separator)
			if len(messages) < 2 {
				log.Println("not processing incoming request")
				return
			}

			targetHandler := string(messages[0])
			handlerFunc, ok := s.handlers[targetHandler]
			if !ok {
				log.Println("handler not found")
				return
			}

			var err error
			ctx := context.Background()
			if s.enableTracing {
				trace := tracer.StartTrace(ctx, "P2PTCP")
				trace.SetTag("remote.addr", conn.RemoteAddr().String())
				trace.Log("request.message", buff)
				defer func() {
					trace.SetError(err)
					logger.LogGreen(s.Name() + " > trace_url: " + tracer.GetTraceURL(ctx))
					trace.Finish()
				}()
				ctx = trace.Context()
			}

			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("%v", r)
				}
				if err != nil {
					conn.Write([]byte(err.Error()))
				}
			}()
			err = handlerFunc(&tcpContextImpl{
				ctx:        ctx,
				conn:       conn,
				bufferSize: s.bufferSize,
				message:    bytes.Join(messages[1:], separator),
			})
		}(conn, s.readAllRequest(conn, buffer))
	}
}

func (s *p2pTCP) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping TCP Server...\x1b[0m")
	defer func() { log.Println("\x1b[33;1mStopping TCP Server:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	s.listener.Close()
}

func (s *p2pTCP) Name() string {
	return string(P2PTCP)
}

func (s *p2pTCP) readAllRequest(conn net.Conn, buff []byte) (b []byte) {

	bytesBuff := new(bytes.Buffer)
	defer bytesBuff.Reset()

	for {
		n, err := conn.Read(buff)
		if err != nil && err != io.EOF {
			return nil
		}
		if n == 0 {
			break
		}
		bytesBuff.Write(buff[0:n])
	}

	return bytesBuff.Bytes()
}
