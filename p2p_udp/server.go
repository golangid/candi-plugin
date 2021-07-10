package p2pudp

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"pkg.agungdp.dev/candi/codebase/factory"
	"pkg.agungdp.dev/candi/logger"
)

type server struct {
	udpConn  *net.UDPConn
	readSize int
	handlers map[string]HandlerFunc
}

// NewP2PUDP init p2p UDP server
func NewP2PUDP(service factory.ServiceFactory, port string) factory.AppServerFactory {
	var srv server
	srv.readSize = 1024
	srv.handlers = make(map[string]HandlerFunc)

	udpAddr, err := net.ResolveUDPAddr("udp4", port)
	if err != nil {
		panic(err)
	}

	srv.udpConn, err = net.ListenUDP("udp4", udpAddr)
	if err != nil {
		panic(err)
	}

	for _, m := range service.GetModules() {
		if h := m.ServerHandler(P2PUDP); h != nil {
			var handlers HandlerGroup
			h.MountHandlers(&handlers)
			for _, handler := range handlers.Handlers {
				srv.handlers[handler.Prefix] = handler.HandlerFunc
				logger.LogYellow(fmt.Sprintf(`[P2P_UDP-SERVER] (route): %-15s  --> (module): "%s"`, `"`+handler.Prefix+`"`, m.Name()))
			}
		}
	}

	fmt.Printf("\x1b[34;1m⇨ P2P UDP server running with %d route\x1b[0m\n\n", len(srv.handlers))
	return &srv
}

func (s *server) Serve() {
	buffer := make([]byte, s.readSize)

	for {
		n, addr, err := s.udpConn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		messages := strings.Split(strings.TrimSpace(string(buffer[0:n])), ":")
		if len(messages) != 2 {
			log.Println("not processing incoming request")
			s.udpConn.WriteToUDP([]byte(EOF), addr)
			continue
		}
		targetHandler := messages[0]
		message := messages[1]
		handlerFunc, ok := s.handlers[targetHandler]
		if !ok {
			log.Println("handler not found")
			s.udpConn.WriteToUDP([]byte("handler '"+targetHandler+"' not found"), addr)
			continue
		}

		ctx := contextImpl{
			ctx:        context.Background(),
			conn:       s.udpConn,
			clientAddr: addr,
			message:    []byte(message),
		}
		if err := handlerFunc(&ctx); err != nil {
			log.Println(err)
		}
	}
}

func (s *server) Shutdown(ctx context.Context) {
	log.Println("\x1b[33;1mStopping UDP Server...\x1b[0m")
	defer func() { log.Println("\x1b[33;1mStopping UDP Server:\x1b[0m \x1b[32;1mSUCCESS\x1b[0m") }()

	s.udpConn.Close()
}

func (s *server) Name() string {
	return string(P2PUDP)
}
