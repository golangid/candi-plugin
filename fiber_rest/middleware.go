package fiberrest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"pkg.agungdp.dev/candi/config/env"
	"pkg.agungdp.dev/candi/logger"
	"pkg.agungdp.dev/candi/tracer"
)

// JaegerTracingMiddleware use jaeger tracing middleware
func JaegerTracingMiddleware(c *fiber.Ctx) error {
	globalTracer := opentracing.GlobalTracer()
	operationName := fmt.Sprintf("%s %s%s", c.Method(), c.BaseURL(), c.Path())

	netHTTPHeader := make(http.Header)
	c.Request().Header.VisitAll(func(key, value []byte) {
		netHTTPHeader.Set(string(key), string(value))
	})

	var span opentracing.Span
	var ctx context.Context
	if spanCtx, err := globalTracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(netHTTPHeader)); err != nil {
		span, ctx = opentracing.StartSpanFromContext(c.Context(), operationName)
		ext.SpanKindRPCServer.Set(span)
	} else {
		span = globalTracer.StartSpan(operationName, opentracing.ChildOf(spanCtx), ext.SpanKindRPCClient)
		ctx = opentracing.ContextWithSpan(c.Context(), span)
	}

	body := c.Body()
	if len(body) < env.BaseEnv().JaegerMaxPacketSize { // limit request body size to 65000 bytes (if higher tracer cannot show root span)
		span.LogKV("request.body", string(body))
	} else {
		span.LogKV("request.body.size", len(body))
	}

	span.SetTag("http.engine", "fiber (fasthttp) version "+fiber.Version)
	span.SetTag("http.method", c.Method())
	span.SetTag("http.raw_url", c.OriginalURL())
	span.SetTag("http.request_header", string(c.Request().Header.RawHeaders()))

	defer func() {
		span.SetTag("http.response_header", string(c.Response().Header.Header()))
		span.SetTag("http.response_code", c.Response().StatusCode())
		resBody := new(bytes.Buffer)
		c.Response().BodyWriteTo(resBody)

		if resBody.Len() < env.BaseEnv().JaegerMaxPacketSize { // limit response body size to 65000 bytes (if higher tracer cannot show root span)
			span.LogKV("response.body", resBody.String())
		} else {
			span.LogKV("response.body.size", resBody.Len())
		}
		span.Finish()

		logger.LogGreen("fiber_rest_api > trace_url: " + tracer.GetTraceURL(ctx))
	}()

	FastHTTPSetContext(ctx, c.Context())
	return c.Next()
}
