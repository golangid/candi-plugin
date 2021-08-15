# Google Cloud Platform PubSub

## Install this plugin in your `candi` service

```sh
$ go get github.com/agungdwiprasetyo/candi-plugin/gcppubsub
```

[OPTIONAL] Set env GOOGLE_APPLICATION_CREDENTIALS=[path to your gcp credentials] for implisit load gcp credential (https://cloud.google.com/docs/authentication/production)

### Register to broker in service config

File `configs/configs.go` in your service

```go
package configs

import (
    "github.com/agungdwiprasetyo/candi-plugin/gcppubsub"
...

// LoadServiceConfigs load selected dependency configuration in this service
func LoadServiceConfigs(baseCfg *config.Config) (deps dependency.Dependency) {
	
		...

		brokerDeps := broker.InitBrokers(
			// another broker,
			// ...
			gcppubsub.SetGCPPubSubBroker(gcppubsub.InitDefaultClient("[your gcp project id]", "[your credentials path]")),
		)

		... 
}
```

### Add in service.go

File `internal/service.go` in your service

```go
package service

import (
    "github.com/agungdwiprasetyo/candi-plugin/gcppubsub"
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
		gcppubsub.NewPubSubWorker(s, "[your-consumer-group-id]"),
	}...)

    ...
}
...
```

### Create delivery handler

Create new file `internal/modules/{{your module}}/delivery/workerhandler/gcp_pubsub_handler.go` in your service

```go
package workerhandler

import (
	"context"
	"encoding/json"

	"example.service/pkg/shared/usecase"

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
	trace, ctx := tracer.StartTraceWithContext(ctx, "DeliveryGCPPubSub:HandleTopic")
	defer trace.Finish()

	log.Printf("message attributes: %+v\n", gcppubsub.GetMessageAttributes(ctx))
	log.Printf("message value: %s\n", message)
	// call usecase
	return nil
}
```

### Register in module.go

File `internal/modules/{{your module}}/module.go` in your service

```go
package examplemodule

import (
	"example.service/internal/modules/examplemodule/delivery/workerhandler"
	"example.service/pkg/shared/usecase"

	"github.com/agungdwiprasetyo/candi-plugin/gcppubsub"

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
			gcppubsub.GoogleCloudPubSub: workerhandler.NewGCPPubSubHandler(usecase.GetSharedUsecase(), deps.GetValidator()),
		},
	}
}

// ...another method
```

### Publisher

Example publish message in your usecase:

```go
package usecase

import (
	"context"

	"github.com/agungdwiprasetyo/candi-plugin/gcppubsub"
	"pkg.agungdp.dev/candi/candishared"
	"pkg.agungdp.dev/candi/codebase/factory/dependency"
	"pkg.agungdp.dev/candi/codebase/interfaces"
)

type usecaseImpl {
	gcpPublisher interfaces.Publisher
}

func NewUsecase(deps dependency.Dependency) Usecase {
	return &usecaseImpl{
		gcpPublisher: deps.GetBroker(gcppubsub.GoogleCloudPubSub).GetPublisher(),
	}
}

func (uc *usecaseImpl) UsecaseToPublishMessage(ctx context.Context) error {
	err := uc.gcpPublisher.PublishMessage(ctx, &candishared.PublisherArgument{
		Topic:  "example-topic",
		Data:   "hello world",
		Header: map[string]interface{}{"key": "value"},
	})
	return err
}
```
