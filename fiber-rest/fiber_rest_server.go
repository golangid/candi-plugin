package fiberrest

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golangid/candi/codebase/factory"
)

type fiberREST struct {
	serverEngine *fiber.App
	service      factory.ServiceFactory
	opt          *option
}

// NewFiberServer create new REST server
func NewFiberServer(service factory.ServiceFactory, opts ...OptionFunc) factory.AppServerFactory {
	server := &fiberREST{
		service: service,
		opt:     getDefaultOption(),
	}

	for _, opt := range opts {
		opt(server.opt)
	}

	server.serverEngine = fiber.New(server.opt.fiberConfig...)
	server.serverEngine.Get("/", server.opt.rootHandler)
	server.serverEngine.Use(cors.New(server.opt.corsConfig))
	root := server.serverEngine.Group(server.opt.rootPath, server.opt.rootMiddlewares...)

	for _, m := range service.GetModules() {
		if h := m.ServerHandler(FiberREST); h != nil {
			h.MountHandlers(root)
		}
	}

	fmt.Printf("\x1b[34;1m⇨ Fiber HTTP server run at port [::]%s\x1b[0m\n\n", server.opt.httpPort)
	return server
}

func (s *fiberREST) Serve() {
	if err := s.serverEngine.Listen(s.opt.httpPort); err != nil {
		panic(err)
	}
}

func (s *fiberREST) Shutdown(ctx context.Context) {
	s.serverEngine.Shutdown()
}

func (s *fiberREST) Name() string {
	return string(FiberREST)
}
