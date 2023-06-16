package fiberrest

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/valyala/fasthttp"
)

const (
	// FiberREST types
	FiberREST types.Server = "fiber-rest"
)

// FastHTTPURLQuery for getter query param by key
type FastHTTPURLQuery struct {
	*fasthttp.Args
}

// Get method
func (f *FastHTTPURLQuery) Get(key string) string {
	return string(f.Peek(key))
}

// ParseGroupHandler parse mount handler param to fiber router
func ParseGroupHandler(i interface{}) fiber.Router {
	return i.(fiber.Router)
}

// URLParam get url param from request context
func URLParam(req *http.Request, key string) string {
	return getRouteContext(req.Context())[key]
}
