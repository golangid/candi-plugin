package stompbroker

import "github.com/golangid/candi/candiutils"

type (
	option struct {
		locker    candiutils.Locker
		debugMode bool
	}

	// OptionFunc type
	OptionFunc func(*option)
)

func getDefaultOption() option {
	return option{
		debugMode: true,
		locker:    &candiutils.NoopLocker{},
	}
}

// SetDebugMode option func
func SetDebugMode(debugMode bool) OptionFunc {
	return func(o *option) {
		o.debugMode = debugMode
	}
}

// SetLocker option func
func SetLocker(locker candiutils.Locker) OptionFunc {
	return func(o *option) {
		o.locker = locker
	}
}
