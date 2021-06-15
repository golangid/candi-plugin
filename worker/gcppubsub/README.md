# Google Cloud Platform PubSub

## Install this plugin in your `candi` service

### Add in service.go

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

### Register in module.go

```go
package examplemodule

import (
	"example.service/internal/modules/examplemodule/delivery/workerhandler"

    "github.com/agungdwiprasetyo/candi-plugin/worker/gcppubsub"

	"pkg.agungdp.dev/candi/codebase/factory/dependency"
	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/codebase/interfaces"
)

type Module struct {
	// ...another delivery handler
	workerHandlers map[types.Worker]interfaces.WorkerHandler
}

func NewModules(deps dependency.Dependency) *Module {
	return &Module{
		workerHandlers: map[types.Worker]interfaces.WorkerHandler{
			// ...another worker handler
			// ...
			gcppubsub.GoogleCloudPubSub: workerhandler.NewGCPPubSubHandler(usecaseUOW.User(), deps.GetValidator()),
		},
	}
}

// ...another method
```

### Create delivery handler

```go
package workerhandler

import (
	"context"
	"encoding/json"
	
	"example.service/internal/modules/examplemodule/delivery/workerhandler"

	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/tracer"
)

// GCPPubSubHandler struct
type GCPPubSubHandler struct {
	uc        usecase.Usecase
	validator interfaces.Validator
}

// NewGCPPubSubHandler constructor
func NewGCPPubSubHandler(uc usecase.Usecase, validator interfaces.Validator) *GCPPubSubHandler {
	return &GCPPubSubHandler{
		uc:        uc,
		validator: validator,
	}
}

// MountHandlers mount handler group
func (h *GCPPubSubHandler) MountHandlers(group *types.WorkerHandlerGroup) {
	group.Add("example-topic", h.handleTopic) // consume topic "example-topic"
}

func (h *GCPPubSubHandler) handleTopic(ctx context.Context, message []byte) error {
	trace := tracer.StartTrace(ctx, "DeliveryGCPPubSub:HandleTopic")
	defer trace.Finish()
	ctx = trace.Context()

	log.Printf("message consumed. message: %s\n", message)
	// call usecase
	return nil
}
```
