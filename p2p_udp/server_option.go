package p2pudp

// ServerOption func type
type ServerOption func(*server)

// ServerSetReadSize option func
func ServerSetReadSize(size int) ServerOption {
	if size <= 0 {
		panic("size must greater than zero")
	}
	return func(s *server) {
		s.readSize = size
	}
}

// ServerSetMaxConcurrentClient option func
func ServerSetMaxConcurrentClient(max int) ServerOption {
	if max <= 0 {
		panic("max must greater than zero")
	}
	return func(s *server) {
		s.maxConcurrentClient = max
	}
}

// ServerDisableTracing option func
func ServerDisableTracing() ServerOption {
	return func(s *server) {
		s.enableTracing = false
	}
}
