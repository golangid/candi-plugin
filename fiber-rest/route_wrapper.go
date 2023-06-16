package fiberrest

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/golangid/candi/codebase/interfaces"
)

type routeWrapper struct {
	router fiber.Router
}

func (r *routeWrapper) Use(middlewares ...func(http.Handler) http.Handler) {
	for _, mw := range middlewares {
		r.router.Use(adaptor.HTTPMiddleware(mw))
	}
}

func (r *routeWrapper) Group(pattern string, middlewares ...func(http.Handler) http.Handler) interfaces.RESTRouter {
	route := r.router.Group(pattern)
	for _, mw := range middlewares {
		route.Use(adaptor.HTTPMiddleware(mw))
	}
	return &routeWrapper{router: route}
}

func (r *routeWrapper) HandleFunc(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.All(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) CONNECT(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Connect(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) DELETE(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Delete(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) GET(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Get(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) HEAD(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Head(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) OPTIONS(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Options(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) PATCH(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Patch(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) POST(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Post(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) PUT(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Put(pattern, WithChainingMiddlewares(h, middlewares...)...)
}

func (r *routeWrapper) TRACE(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Trace(pattern, WithChainingMiddlewares(h, middlewares...)...)
}
