package fiberrest

import "github.com/valyala/fasthttp"

// FastHTTPURLQuery for getter query param by key
type FastHTTPURLQuery struct {
	*fasthttp.Args
}

// Get method
func (f *FastHTTPURLQuery) Get(key string) string {
	return string(f.Peek(key))
}
