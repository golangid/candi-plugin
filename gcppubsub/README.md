# Google Cloud Platform PubSub

## Install this plugin in your `candi` service

```sh
$ go get github.com/golangid/candi-plugin/gcppubsub
```

[OPTIONAL] Set env GOOGLE_APPLICATION_CREDENTIALS=[path to your gcp credentials] for implisit load gcp credential (https://cloud.google.com/docs/authentication/production)

### Register to broker in service config

File `configs/configs.go` in your service

```go
package configs

import (
    "github.com/golangid/candi-plugin/gcppubsub"
...

// LoadServiceConfigs load selected dependency configuration in this service
func LoadServiceConfigs(baseCfg *config.Config) (deps dependency.Dependency) {
	
		...

		brokerDeps := broker.InitBrokers(
			// another broker,
			// ...
			gcppubsub.NewGCPPubSubBroker(
				gcppubsub.BrokerSetClient(gcppubsub.InitDefaultClient("[your gcp project id]", "[your credentials path]")),
			),
		)

		... 
}
```

### Init worker in app_factory.go for consume message

File `configs/app_factory.go` in your service

```go
package configs

import (
	"github.com/golangid/candi-plugin/gcppubsub"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/appfactory"
	"github.com/golangid/candi/config/env"
)

func InitAppFromEnvironmentConfig(service factory.ServiceFactory) (apps []factory.AppServerFactory) {

	...


	apps = append(apps, gcppubsub.NewPubSubWorker(service, service.GetDependency().GetBroker(gcppubsub.GoogleCloudPubSub), "[your-consumer/subscriber-group-id]"))
	return
}
```

### Create delivery handler

Create new file `internal/modules/{{your module}}/delivery/workerhandler/gcp_pubsub_handler.go` in your service

```go
package workerhandler

import (
	"context"
	"encoding/json"

	"example.service/pkg/shared/usecase"

	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/tracer"
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

func (h *GCPPubSubHandler) handleTopic(eventContext *candishared.EventContext) error {
	trace, _ := tracer.StartTraceWithContext(eventContext.Context(), "DeliveryGCPPubSub:HandleTopic")
	defer trace.Finish()

	log.Printf("message attributes: %+v\n", eventContext.Header())
	log.Printf("message value: %s\n", eventContext.Message())
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

	"github.com/golangid/candi-plugin/gcppubsub"
	"github.com/golangid/candi/codebase/factory/dependency"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
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

	"github.com/golangid/candi-plugin/gcppubsub"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/factory/dependency"
	"github.com/golangid/candi/codebase/interfaces"
)

type usecaseImpl {
	deps dependency.Dependency
}

func NewUsecase(deps dependency.Dependency) Usecase {
	return &usecaseImpl{
		deps: deps,
	}
}

func (uc *usecaseImpl) UsecaseToPublishMessage(ctx context.Context) error {
	err := uc.deps.GetBroker(gcppubsub.GoogleCloudPubSub).GetPublisher().PublishMessage(ctx, &candishared.PublisherArgument{
		Topic:		"example-topic",
		Message:	"hello world",
		Header:		map[string]interface{}{"key": "value"},
	})
	return err
}
```
