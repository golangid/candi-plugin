package circuitbreaker

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gojektech/heimdall"
	"github.com/gojektech/heimdall/hystrix"
)

type (
	hystrixOption struct {
		breakerName       string
		timeout           time.Duration
		retries           int
		sleepBetweenRetry time.Duration
		tlsConfig         *tls.Config

		additionalOption []hystrix.Option
	}

	// HystrixOption func type
	HystrixOption func(*hystrixOption)
)

// HystrixSetRetries option func
func HystrixSetRetries(retries int) HystrixOption {
	return func(h *hystrixOption) {
		h.retries = retries
	}
}

// HystrixSetSleepBetweenRetry option func
func HystrixSetSleepBetweenRetry(sleepBetweenRetry time.Duration) HystrixOption {
	return func(h *hystrixOption) {
		h.sleepBetweenRetry = sleepBetweenRetry
	}
}

// HystrixSetTLS option func
func HystrixSetTLS(tlsConfig *tls.Config) HystrixOption {
	return func(h *hystrixOption) {
		h.tlsConfig = tlsConfig
	}
}

// HystrixSetTimeout option func
func HystrixSetTimeout(timeout time.Duration) HystrixOption {
	return func(h *hystrixOption) {
		h.timeout = timeout
	}
}

// HystrixSetBreakerName option func
func HystrixSetBreakerName(breakerName string) HystrixOption {
	return func(h *hystrixOption) {
		h.breakerName = breakerName
	}
}

// NewHystrixCircuitBreaker create new circuit breaker using hystrix
func NewHystrixCircuitBreaker(opts ...HystrixOption) interface {
	Do(*http.Request) (*http.Response, error)
} {
	clientOpt := new(hystrixOption)

	// set default value
	clientOpt.retries = 5
	clientOpt.sleepBetweenRetry = 500 * time.Millisecond
	clientOpt.timeout = 10 * time.Second
	clientOpt.breakerName = "default"
	for _, opt := range opts {
		opt(clientOpt)
	}

	// define a maximum jitter interval
	maximumJitterInterval := 10 * time.Millisecond
	// create a backoff
	backoff := heimdall.NewConstantBackoff(clientOpt.sleepBetweenRetry, maximumJitterInterval)
	// create a new retry mechanism with the backoff
	retrier := heimdall.NewRetrier(backoff)

	hystrixClientOpt := []hystrix.Option{
		hystrix.WithHTTPTimeout(clientOpt.timeout),
		hystrix.WithHystrixTimeout(clientOpt.timeout),
		hystrix.WithRetrier(retrier),
		hystrix.WithRetryCount(clientOpt.retries),
		hystrix.WithCommandName(clientOpt.breakerName),
		hystrix.WithFallbackFunc(func(err error) error {
			return err
		}),
	}
	hystrixClientOpt = append(hystrixClientOpt, clientOpt.additionalOption...)
	if clientOpt.tlsConfig != nil {
		hystrixClientOpt = append(hystrixClientOpt, hystrix.WithHTTPClient(&http.Client{
			Transport: &http.Transport{TLSClientConfig: clientOpt.tlsConfig},
		}))
	}

	return hystrix.NewClient(hystrixClientOpt...)
}
