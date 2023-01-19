package fiberrest

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/golangid/candi/config/env"
	"github.com/golangid/candi/logger"
	"github.com/golangid/candi/tracer"
)

// JaegerTracingMiddleware use jaeger tracing middleware
func JaegerTracingMiddleware(c *fiber.Ctx) error {
	operationName := fmt.Sprintf("%s %s%s", c.Method(), c.BaseURL(), c.Path())

	netHTTPHeader := make(http.Header)
	c.Request().Header.VisitAll(func(key, value []byte) {
		netHTTPHeader.Set(string(key), string(value))
	})

	trace, ctx := tracer.StartTraceFromHeader(c.Context(), operationName, c.GetReqHeaders())
	defer func() {
		trace.Log("http.response_header", string(c.Response().Header.Header()))
		trace.SetTag("http.response_code", c.Response().StatusCode())

		resBody := c.Response().Body()
		if len(resBody) < env.BaseEnv().JaegerMaxPacketSize { // limit response body size to 65000 bytes (if higher tracer cannot show root span)
			trace.Log("response.body", resBody)
		} else {
			trace.Log("response.body.size", resBody)
		}
		trace.Finish()
		logger.LogGreen("fiber_rest_api > trace_url: " + tracer.GetTraceURL(ctx))
	}()

	body := c.Body()
	if len(body) < env.BaseEnv().JaegerMaxPacketSize { // limit request body size to 65000 bytes (if higher tracer cannot show root span)
		trace.Log("request.body", string(body))
	} else {
		trace.Log("request.body.size", len(body))
	}

	trace.SetTag("http.engine", "fiber (fasthttp) version "+fiber.Version)
	trace.SetTag("http.method", c.Method())
	trace.SetTag("http.url_path", c.Path())
	trace.SetTag("http.full_url", c.OriginalURL())
	trace.Log("http.request_header", string(c.Request().Header.RawHeaders()))

	FastHTTPSetContext(ctx, c.Context())
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
