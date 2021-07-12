package p2p

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"

	"pkg.agungdp.dev/candi/codebase/factory"
	"pkg.agungdp.dev/candi/logger"
	"pkg.agungdp.dev/candi/tracer"
)

type p2pUDP struct {
	option

	udpConn   *net.UDPConn
	handlers  map[string]HandlerFunc
	semaphore chan struct{}
}

// NewP2PUDP init p2p UDP network
func NewP2PUDP(service factory.ServiceFactory, port string, opts ...ServerOption) factory.AppServerFactory {
	srv := new(p2pUDP)
	srv.bufferSize = 1024
	srv.maxConcurrentClient = 100
	srv.enableTracing = true

	for _, opt := range opts {
		opt(&srv.option)
	}

	srv.semaphore = make(chan struct{}, srv.maxConcurrentClient)
	srv.handlers = make(map[string]HandlerFunc)

	udpAddr, err := net.ResolveUDPAddr("udp4", port)
	if err != nil {
		panic(err)
	}

	srv.udpConn, err = net.ListenUDP("udp4", udpAddr)
	if err != nil {
		panic(err)
	}
	srv.udpConn.SetWriteBuffer(srv.bufferSize)

	for _, m := range service.GetModules() {
		if h := m.ServerHandler(P2PUDP); h != nil {
			var handlers HandlerGroup
			h.MountHandlers(&handlers)
			for _, handler := range handlers.Handlers {
				srv.handlers[handler.Prefix] = handler.HandlerFunc
				logger.LogYellow(fmt.Sprintf(`[P2P_UDP] (route): %-15s  --> (module): "%s"`, `"`+handler.Prefix+`"`, m.Name()))
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ P2P UDP running with %d route and port [::]%s\x1b[0m\n\n", len(srv.handlers), port)
	return srv
}

func (s *p2pUDP) Serve() {
	buffer := make([]byte, s.bufferSize)

	for {
		n, addr, err := s.udpConn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		s.semaphore <- struct{}{}
		go func(clientAddr *net.UDPAddr, buff []byte) {
			defer func() { <-s.semaphore }()

			messages := bytes.Split(bytes.TrimSpace(buff), []byte(":"))
			if len(messages) < 2 {
				log.Println("not processing incoming request")
				s.udpConn.WriteToUDP([]byte(EOF), clientAddr)
				return
			}

			targetHandler := string(messages[0])
			handlerFunc, ok := s.handlers[targetHandler]
			if !ok {
				log.Println("handler not found")
				s.udpConn.WriteToUDP([]byte("handler '"+targetHandler+"' not found"), addr)
				return
			}

			var err error
			ctx := context.Background()
			if s.enableTracing {
				trace := tracer.StartTrace(ctx, "P2PUDP")
				trace.SetTag("client.network", clientAddr.Network())
				trace.SetTag("client.addr", clientAddr.String())
				trace.Log("request_message", buff)
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
					s.udpConn.WriteToUDP([]byte(err.Error()), addr)
				}
			}()
			err = handlerFunc(&udpContextImpl{
				ctx:        ctx,
				conn:       s.udpConn,
				clientAddr: clientAddr,
				message:    bytes.Join(messages[1:], []byte(":")),
			})
		}(addr, buffer[0:n])
	}
}

func (s *p2pUDP) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping UDP Server...\x1b[0m")
	defer func() { log.Println("\x1b[33;1mStopping UDP Server:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	s.udpConn.Close()
}

func (s *p2pUDP) Name() string {
	return string(P2PUDP)
}
