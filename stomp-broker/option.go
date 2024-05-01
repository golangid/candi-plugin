package stompbroker

import (
	"github.com/golangid/candi/candiutils"
	"github.com/golangid/candi/codebase/interfaces"
)

type (
	option struct {
		locker    interfaces.Locker
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
func SetLocker(locker interfaces.Locker) OptionFunc {
	return func(o *option) {
		o.locker = locker
	}
}
