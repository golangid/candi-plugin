package fiberrest

import (
	"errors"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	graphqlserver "github.com/golangid/candi/codebase/app/graphql_server"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/wrapper"
)

type (
	option struct {
		fiberConfig         fiber.Config
		corsConfig          cors.Config
		rootMiddlewares     []func(*fiber.Ctx) error
		rootHandler         func(*fiber.Ctx) error
		httpPort            uint16
		rootPath            string
		debugMode           bool
		jaegerMaxPacketSize int

		includeGraphQL bool
		graphqlOption  graphqlserver.Option
	}

	// OptionFunc type
	OptionFunc func(*option)
)

func defaultErrHandler(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}
	return ctx.JSON(wrapper.HTTPResponse{
		Code: code, Success: false, Message: err.Error(),
	})
}

func getDefaultOption() *option {
	return &option{
		httpPort:  8000,
		rootPath:  "",
		debugMode: true,
		rootMiddlewares: []func(*fiber.Ctx) error{
			RecoverMiddleware,
			JaegerTracingMiddleware,
			adaptor.HTTPMiddleware(wrapper.HTTPMiddlewareLog(env.BaseEnv().DebugMode, os.Stdout)),
		},
		rootHandler: adaptor.HTTPHandlerFunc(wrapper.HTTPHandlerDefaultRoot),
		corsConfig:  cors.Config{},
		fiberConfig: fiber.Config{
			ErrorHandler: defaultErrHandler,
		},
	}
}

// SetHTTPPort option func
func SetHTTPPort(port uint16) OptionFunc {
	return func(o *option) {
		o.httpPort = port
	}
}

// SetRootPath option func
func SetRootPath(rootPath string) OptionFunc {
	return func(o *option) {
		o.rootPath = rootPath
	}
}

// SetRootHTTPHandler option func
func SetRootHTTPHandler(rootHandler func(*fiber.Ctx) error) OptionFunc {
	return func(o *option) {
		o.rootHandler = rootHandler
	}
}

// SetDebugMode option func
func SetDebugMode(debugMode bool) OptionFunc {
	return func(o *option) {
		o.debugMode = debugMode
	}
}

// SetRootMiddlewares option func
func SetRootMiddlewares(middlewares ...func(*fiber.Ctx) error) OptionFunc {
	return func(o *option) {
		o.rootMiddlewares = middlewares
	}
}

// AddRootMiddlewares option func
func AddRootMiddlewares(middleware func(*fiber.Ctx) error) OptionFunc {
	return func(o *option) {
		o.rootMiddlewares = append(o.rootMiddlewares, middleware)
	}
}

// SetFiberConfig option func
func SetFiberConfig(cfg fiber.Config) OptionFunc {
	return func(o *option) {
		if cfg.ErrorHandler == nil {
			cfg.ErrorHandler = defaultErrHandler
		}
		o.fiberConfig = cfg
	}
}

func SetCorsConfig(cfg cors.Config) OptionFunc {
	return func(o *option) {
		o.corsConfig = cfg
	}
}

// SetJaegerMaxPacketSize option func
func SetJaegerMaxPacketSize(max int) OptionFunc {
	return func(o *option) {
		o.jaegerMaxPacketSize = max
	}
}

// SetIncludeGraphQL option func
func SetIncludeGraphQL(includeGraphQL bool) OptionFunc {
	return func(o *option) {
		o.includeGraphQL = includeGraphQL
	}
}

// AddGraphQLOption option func
func AddGraphQLOption(opts ...graphqlserver.OptionFunc) OptionFunc {
	return func(o *option) {
		o.graphqlOption.RootPath = "/graphql"
		for _, opt := range opts {
			opt(&o.graphqlOption)
		}
	}
}
