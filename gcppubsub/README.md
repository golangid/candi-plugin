# Google Cloud Platform PubSub

## Install this plugin in your `candi` service

```sh
$ go get github.com/agungdwiprasetyo/candi-plugin/gcppubsub
```

Please set env GOOGLE_APPLICATION_CREDENTIALS=[path to your gcp credentials] for implisit load gcp credential (https://cloud.google.com/docs/authentication/production)

### Add in service.go

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
		gcppubsub.NewPubSubWorker(s, gcppubsub.InitDefaultClient("[your gcp project id]", "[your credentials path]"), "[your-consumer-group-id]"),
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

### Create delivery handler

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


### Publisher

Example:

```go
func publish(){
	pub := gcppubsub.NewPublisher(gcppubsub.InitDefaultClient("[your gcp project id]", "[your credentials path]"))
	if err := pub.PublishMessage(context.Background(), &candishared.PublisherArgument{
		Topic:  "example-topic",
		Data:   "hello world",
		Header: map[string]interface{}{"key": "value"},
	}); err != nil {
		panic(err)
	}
}
```
