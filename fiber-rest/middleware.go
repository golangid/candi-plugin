package fiberrest

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/tracer"
)

// JaegerTracingMiddleware use jaeger tracing middleware
func JaegerTracingMiddleware(c *fiber.Ctx) error {
	if candihelper.StringInSlice(c.Path(), []string{"/", "/graphql"}) {
		return c.Next()
	}

	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	trace, ctx := tracer.StartTraceFromHeader(c.Context(), "fasthttp.request", headers)
	defer func() {
		trace.Log("http.response_header", string(c.Response().Header.Header()))
		trace.SetTag("http.status_code", c.Response().StatusCode())

		resBody := c.Response().Body()
		if len(resBody) < env.BaseEnv().JaegerMaxPacketSize {
			trace.Log("response.body", resBody)
		} else {
			trace.Log("response.body.size", candihelper.TransformSizeToByte(uint64(len(resBody))))
		}
		trace.Finish()
	}()

	r := c.Route()
	trace.SetTag("resource.name", r.Method+" "+c.Path())

	body := c.Body()
	if len(body) < env.BaseEnv().JaegerMaxPacketSize {
		trace.Log("request.body", string(body))
	} else {
		trace.Log("request.body.size", candihelper.TransformSizeToByte(uint64(len(body))))
	}

	trace.SetTag("http.engine", "fiber (fasthttp) version "+fiber.Version)
	trace.SetTag("http.method", c.Method())
	trace.SetTag("http.url_path", c.Path())
	for key := range c.GetReqHeaders() {
		trace.SetTag("http.headers."+key, c.Get(key))
	}
	trace.Log("http.full_url", c.OriginalURL())
	trace.Log("http.request_header", string(c.Request().Header.RawHeaders()))

	adaptor.CopyContextToFiberContext(ctx, c.Context())
	return c.Next()
}

func RecoverMiddleware(c *fiber.Ctx) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewHTTPResponse(http.StatusInternalServerError, fmt.Sprintf("panic: %v", r)).JSON(c.Response())
		}
	}()

	return c.Next()
}

// WithChainingMiddlewares chaining middlewares
func WithChainingMiddlewares(handlerFunc http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) (fiberHandlers []fiber.Handler) {
	for _, mw := range middlewares {
		fiberHandlers = append(fiberHandlers, adaptor.HTTPMiddleware(mw))
	}
	fiberHandlers = append(fiberHandlers, WrapHTTPHandlerFunc(handlerFunc))
	return fiberHandlers
}
