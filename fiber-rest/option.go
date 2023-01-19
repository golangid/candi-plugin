package fiberrest

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type (
	option struct {
		fiberConfig     []fiber.Config
		corsConfig      cors.Config
		rootMiddlewares []func(*fiber.Ctx) error
		rootHandler     func(*fiber.Ctx) error
		httpPort        string
		rootPath        string
		debugMode       bool
	}

	// OptionFunc type
	OptionFunc func(*option)
)

func getDefaultOption() *option {
	return &option{
		httpPort:        ":8000",
		rootPath:        "",
		debugMode:       true,
		rootMiddlewares: []func(*fiber.Ctx) error{RecoverMiddleware, JaegerTracingMiddleware},
		rootHandler: func(c *fiber.Ctx) error {
			c.Response().BodyWriter().Write([]byte("REST Server up and running"))
			return nil
		},
		corsConfig: cors.Config{},
	}
}

// SetHTTPPort option func
func SetHTTPPort(port uint16) OptionFunc {
	return func(o *option) {
		o.httpPort = fmt.Sprintf(":%d", port)
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
func SetFiberConfig(cfg ...fiber.Config) OptionFunc {
	return func(o *option) {
		o.fiberConfig = cfg
	}
}

func SetCorsConfig(cfg cors.Config) OptionFunc {
	return func(o *option) {
		o.corsConfig = cfg
	}
}
