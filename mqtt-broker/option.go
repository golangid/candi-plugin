package mqttbroker

type (
	option struct {
		maxGoroutines int
		debugMode     bool
	}

	// OptionFunc type
	OptionFunc func(*option)
)

func getDefaultOption() option {
	opt := option{
		maxGoroutines: 10,
		debugMode:     true,
	}
	return opt
}

// SetMaxGoroutines option func
func SetMaxGoroutines(maxGoroutines int) OptionFunc {
	return func(o *option) {
		o.maxGoroutines = maxGoroutines
	}
}

// SetDebugMode option func
func SetDebugMode(debugMode bool) OptionFunc {
	return func(o *option) {
		o.debugMode = debugMode
	}
}
