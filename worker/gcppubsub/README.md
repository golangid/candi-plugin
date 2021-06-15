# Google Cloud Platform PubSub

## Add to your `candi` service
```go
package service

import (
    "github.com/agungdwiprasetyo/candi-plugin/worker/gcppubsub"
...

// Service model
type Service struct {
	applications []factory.AppServerFactory
...

// NewService in this service
func NewService(cfg *config.Config) factory.ServiceFactory {
	...

	s := &Service{
        ...

    // Add custom application runner, must implement `factory.AppServerFactory` methods
	s.applications = append(s.applications, []factory.AppServerFactory{
		// customApplication
		gcppubsub.NewPubSubWorker(s, "[your gcp project id]", "[your-consumer-group-id]"),
	}...)

    ...
}
...
```
