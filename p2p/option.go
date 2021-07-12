package p2p

type option struct {
	bufferSize          int
	maxConcurrentClient int
	enableTracing       bool
}

// ServerOption func type
type ServerOption func(*option)

// ServerSetBufferSize option func
func ServerSetBufferSize(size int) ServerOption {
	if size <= 0 {
		panic("size must greater than zero")
	}
	return func(o *option) {
		o.bufferSize = size
	}
}

// ServerSetMaxConcurrentClient option func
func ServerSetMaxConcurrentClient(max int) ServerOption {
	if max <= 0 {
		panic("max must greater than zero")
	}
	return func(o *option) {
		o.maxConcurrentClient = max
	}
}

// ServerDisableTracing option func
func ServerDisableTracing() ServerOption {
	return func(o *option) {
		o.enableTracing = false
	}
}
