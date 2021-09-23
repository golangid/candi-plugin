package fiberrest

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/golangid/candi/codebase/factory"
)

type fiberREST struct {
	serverEngine *fiber.App
	service      factory.ServiceFactory
	httpPort     string
}

// NewFiberServer create new REST server
func NewFiberServer(service factory.ServiceFactory, httpPort string, rootMiddleware ...func(*fiber.Ctx) error) factory.AppServerFactory {
	server := &fiberREST{
		serverEngine: fiber.New(),
		service:      service,
		httpPort:     httpPort,
	}

	root := server.serverEngine.Group("/", rootMiddleware...)
	for _, m := range service.GetModules() {
		if h := m.ServerHandler(FiberREST); h != nil {
			h.MountHandlers(root)
		}
	}

	fmt.Printf("\x1b[34;1m⇨ Fiber HTTP server run at port [::]%s\x1b[0m\n\n", server.httpPort)
	return server
}

func (s *fiberREST) Serve() {
	if err := s.serverEngine.Listen(s.httpPort); err != nil {
		panic(err)
	}
}

func (s *fiberREST) Shutdown(ctx context.Context) {
	// h.serverEngine.Shutdown()
}

func (s *fiberREST) Name() string {
	return string(FiberREST)
}
