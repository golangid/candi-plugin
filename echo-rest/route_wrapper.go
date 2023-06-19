package echorest

import (
	"net/http"
	"strings"

	"github.com/golangid/candi/codebase/interfaces"
	"github.com/labstack/echo"
)

type routeWrapper struct {
	router *echo.Group
}

func (r *routeWrapper) Use(middlewares ...func(http.Handler) http.Handler) {
	for _, mw := range middlewares {
		r.router.Use(echo.WrapMiddleware(mw))
	}
}

func (r *routeWrapper) Group(pattern string, middlewares ...func(http.Handler) http.Handler) interfaces.RESTRouter {
	route := r.router.Group(pattern)
	for _, mw := range middlewares {
		route.Use(echo.WrapMiddleware(mw))
	}
	return &routeWrapper{router: route}
}

func (r *routeWrapper) HandleFunc(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.Any(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) CONNECT(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.CONNECT(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) DELETE(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.DELETE(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) GET(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.GET(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) HEAD(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.HEAD(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) OPTIONS(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.OPTIONS(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) PATCH(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.PATCH(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) POST(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.POST(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) PUT(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.PUT(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

func (r *routeWrapper) TRACE(pattern string, h http.HandlerFunc, middlewares ...func(http.Handler) http.Handler) {
	r.router.TRACE(strings.TrimSuffix(pattern, "/"), WrapHandler(h), transformMiddlewares()...)
}

// transformMiddlewares chaining middlewares
func transformMiddlewares(middlewares ...func(http.Handler) http.Handler) (mFuncs []echo.MiddlewareFunc) {
	for _, mw := range middlewares {
		mFuncs = append(mFuncs, EchoWrapMiddleware(mw))
	}
	return mFuncs
}

// WrapHandler wraps `http.Handler` into `echo.HandlerFunc`.
func WrapHandler(h http.Handler) echo.HandlerFunc {
	return func(c echo.Context) error {
		params := make(map[string]string, len(c.ParamNames()))
		for i, name := range c.ParamNames() {
			params[name] = c.ParamValues()[i]
		}

		req := c.Request()
		ctx := setToContext(req.Context(), params)
		req.WithContext(ctx)
		c.SetRequest(req)
		h.ServeHTTP(c.Response(), req)
		return nil
	}
}
