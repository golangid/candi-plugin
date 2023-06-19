package fiberrest

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golangid/candi/candihelper"
	graphqlserver "github.com/golangid/candi/codebase/app/graphql_server"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
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

	server.serverEngine = fiber.New(server.opt.fiberConfig)
	server.serverEngine.Get("/", server.opt.rootHandler)
	server.serverEngine.Use(cors.New(server.opt.corsConfig))
	root := server.serverEngine.Group(server.opt.rootPath, server.opt.rootMiddlewares...)

	rw := &routeWrapper{router: root}
	for _, m := range service.GetModules() {
		// for default candi rest handler
		if h := m.RESTHandler(); h != nil {
			h.Mount(rw)
		}

		// additional if still using fiber routing
		if h := m.ServerHandler(FiberREST); h != nil {
			h.MountHandlers(root)
		}
	}

	for _, routes := range server.serverEngine.GetRoutes() {
		if candihelper.StringInSlice(routes.Path, []string{"/", "/memstats"}) {
			continue
		}
		if candihelper.StringInSlice(routes.Method, []string{http.MethodHead, http.MethodConnect, http.MethodOptions, http.MethodTrace}) {
			continue
		}
		logger.LogGreen(fmt.Sprintf("[REST-ROUTE] %-6s %-30s", routes.Method, strings.TrimSuffix(routes.Path, "/")))
	}

	// inject graphql handler to rest server
	if server.opt.includeGraphQL {
		gqlOpt := server.opt.graphqlOption
		gqlRootPath := gqlOpt.RootPath
		gqlOpt.RootPath = strings.Trim(server.opt.rootPath, "/") + gqlRootPath
		graphqlHandler := graphqlserver.ConstructHandlerFromService(service, gqlOpt)

		root.All(gqlRootPath, adaptor.HTTPHandlerFunc(graphqlHandler.ServeGraphQL()))
		root.Get(gqlRootPath+"/playground", adaptor.HTTPHandlerFunc(http.HandlerFunc(graphqlHandler.ServePlayground)))
		root.Get(gqlRootPath+"/voyager", adaptor.HTTPHandlerFunc(http.HandlerFunc(graphqlHandler.ServeVoyager)))
	}

	fmt.Printf("\x1b[34;1m⇨ Fiber HTTP server run at port [::]:%d\x1b[0m\n\n", server.opt.httpPort)
	return server
}

func (s *fiberREST) Serve() {
	if err := s.serverEngine.Listen(fmt.Sprintf(":%d", s.opt.httpPort)); err != nil {
		panic(err)
	}
}

func (s *fiberREST) Shutdown(ctx context.Context) {
	s.serverEngine.ShutdownWithContext(ctx)
}

func (s *fiberREST) Name() string {
	return string(FiberREST)
}

// SetupFiberServer setup fiber rest server with default config
func SetupFiberServer(service factory.ServiceFactory, opts ...OptionFunc) factory.AppServerFactory {
	restOptions := []OptionFunc{
		SetHTTPPort(env.BaseEnv().HTTPPort),
		SetRootPath(env.BaseEnv().HTTPRootPath),
		SetIncludeGraphQL(env.BaseEnv().UseGraphQL),
		SetDebugMode(env.BaseEnv().DebugMode),
		SetJaegerMaxPacketSize(env.BaseEnv().JaegerMaxPacketSize),
	}
	if env.BaseEnv().UseGraphQL {
		restOptions = append(restOptions, AddGraphQLOption(
			graphqlserver.SetDisableIntrospection(env.BaseEnv().GraphQLDisableIntrospection),
			graphqlserver.SetHTTPPort(env.BaseEnv().HTTPPort),
		))
	}
	restOptions = append(restOptions, opts...)
	return NewFiberServer(service, restOptions...)
}
