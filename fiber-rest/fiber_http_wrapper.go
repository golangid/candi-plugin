package fiberrest

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/golangid/candi/candishared"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var (
	fiberURLParamCtxKey = candishared.ContextKey("fiberURLParamCtxKey")
)

func setToContext(ctx context.Context, value map[string]string) context.Context {
	return context.WithValue(ctx, fiberURLParamCtxKey, value)
}

func getRouteContext(ctx context.Context) map[string]string {
	val, _ := ctx.Value(fiberURLParamCtxKey).(map[string]string)
	return val
}

// WrapHTTPHandlerFunc wraps net/http handler func to fiber handler with param value to context
func WrapHTTPHandlerFunc(h http.HandlerFunc) fiber.Handler {
	return func(c *fiber.Ctx) error {
		handler := fasthttpadaptor.NewFastHTTPHandler(h)
		ctx := c.Context()
		if params := c.AllParams(); len(params) > 0 {
			ctx.SetUserValue(fiberURLParamCtxKey, params)
		}
		handler(ctx)
		return nil
	}
}
