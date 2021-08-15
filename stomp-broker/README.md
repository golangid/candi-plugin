# STOMP Broker

## Install this plugin in your `candi` service

```sh
$ go get github.com/agungdwiprasetyo/candi-plugin/stomp-broker
```

### Register to broker in service config

File `configs/configs.go` in your service

```go
package configs

import (
    "github.com/agungdwiprasetyo/candi-plugin/stomp-broker"
...

// LoadServiceConfigs load selected dependency configuration in this service
func LoadServiceConfigs(baseCfg *config.Config) (deps dependency.Dependency) {
	
		...

		brokerDeps := broker.InitBrokers(
			// another broker,
			// ...
			stompbroker.SetSTOMPBroker(stompbroker.InitDefaultConnection("[broker host]", "[username]", "[password]")),
		)

		... 
}
```

### Add in service.go

File `internal/service.go` in your service

```go
package service

import (
    "github.com/agungdwiprasetyo/candi-plugin/stomp-broker"
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
		stompbroker.NewSTOMPWorker(s),
	}...)

    ...
}
...
```

### Create delivery handler

Create new file `internal/modules/{{your module}}/delivery/workerhandler/stomp_handler.go` in your service

```go
package workerhandler

import (
	"context"
	"encoding/json"

	"example.service/pkg/shared/usecase"

	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/tracer"
)

// STOMPHandler struct
type STOMPHandler struct {
	uc        usecase.Usecase
	validator interfaces.Validator
}

// NewSTOMPHandler constructor
func NewSTOMPHandler(uc usecase.Usecase, validator interfaces.Validator) *STOMPHandler {
	return &STOMPHandler{
		uc:        uc,
		validator: validator,
	}
}

// MountHandlers mount handler group
func (h *STOMPHandler) MountHandlers(group *types.WorkerHandlerGroup) {
	group.Add("example-topic", h.handleTopic) // consume topic "example-topic"
}

func (h *STOMPHandler) handleTopic(ctx context.Context, message []byte) error {
	trace, ctx := tracer.StartTraceWithContext(ctx, "DeliverySTOMPWorker:HandleTopic")
	defer trace.Finish()

	log.Printf("message value: %s\n", message)
	// call usecase
	return nil
}
```

### Register in module.go

```go
package examplemodule

import (
	"example.service/internal/modules/{{your module}}/delivery/workerhandler"
	"example.service/pkg/shared/usecase"

	"github.com/agungdwiprasetyo/candi-plugin/stomp-broker"

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
			stompbroker.STOMPBroker: workerhandler.NewSTOMPHandler(usecase.GetSharedUsecase(), deps.GetValidator()),
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

	"github.com/agungdwiprasetyo/candi-plugin/stomp-broker"
	"pkg.agungdp.dev/candi/candishared"
	"pkg.agungdp.dev/candi/codebase/factory/dependency"
	"pkg.agungdp.dev/candi/codebase/interfaces"
)

type usecaseImpl {
	stompPublisher interfaces.Publisher
}

func NewUsecase(deps dependency.Dependency) Usecase {
	return &usecaseImpl{
		stompPublisher: deps.GetBroker(stompbroker.STOMPBroker).GetPublisher(),
	}
}

func (uc *usecaseImpl) UsecaseToPublishMessage(ctx context.Context) error {
	err := uc.stompPublisher.PublishMessage(ctx, &candishared.PublisherArgument{
		Topic:  "example-topic",
		Data:   "hello world",
		Header: map[string]interface{}{"key": "value"},
	})
	return err
}
```
