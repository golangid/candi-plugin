# MQTT Broker PubSub

## Install this plugin in your `candi` service

```sh
$ go get github.com/golangid/candi-plugin/mqtt-broker
```

### Register to broker in service config

File `configs/configs.go` in your service

```go
package configs

import (
    mqttbroker "github.com/golangid/candi-plugin/mqtt-broker"
...

// LoadServiceConfigs load selected dependency configuration in this service
func LoadServiceConfigs(baseCfg *config.Config) (deps dependency.Dependency) {
	
		...

		brokerDeps := broker.InitBrokers(
			// another broker,
			// ...
			mqttbroker.NewMQTTBroker(mqtt.NewClientOptions().
				AddBroker("tcp://127.0.0.1:1883").
				SetClientID("examples").
				SetCleanSession(false).
				SetUsername("admin").
				SetPassword("password").
				SetAutoReconnect(true).
				SetConnectRetry(true),
				mqttbroker.BrokerSetPublisherQOS(2),
				mqttbroker.BrokerSetSubscriberQOS(2),
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
	mqttbroker "github.com/golangid/candi-plugin/mqtt-broker"
	"github.com/golangid/candi/codebase/factory"
	"github.com/golangid/candi/codebase/factory/appfactory"
	"github.com/golangid/candi/config/env"
)

func InitAppFromEnvironmentConfig(service factory.ServiceFactory) (apps []factory.AppServerFactory) {

	...


	apps = append(apps, mqttbroker.NewMQTTSubscriber(service, service.GetDependency().GetBroker(mqttbroker.MQTTBroker)))
	return
}
```

### Create delivery handler

Create new file `internal/modules/{{your module}}/delivery/workerhandler/mqtt_handler.go` in your service

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

// MQTTHandler struct
type MQTTHandler struct {
	uc        usecase.Usecase
	validator interfaces.Validator
}

// NewMQTTHandler constructor
func NewMQTTHandler(uc usecase.Usecase, validator interfaces.Validator) *MQTTHandler {
	return &MQTTHandler{
		uc:        uc,
		validator: validator,
	}
}

// MountHandlers mount handler group
func (h *MQTTHandler) MountHandlers(group *types.WorkerHandlerGroup) {
	group.Add("example-topic", h.handleTopic) // consume topic "example-topic"
}

func (h *MQTTHandler) handleTopic(eventContext *candishared.EventContext) error {
	trace, _ := tracer.StartTraceWithContext(eventContext.Context(), "DeliveryMQTT:HandleTopic")
	defer trace.Finish()

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

	mqttbroker "github.com/golangid/candi-plugin/mqtt-broker"
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
			mqttbroker.MQTTBroker: workerhandler.NewMQTTHandler(usecase.GetSharedUsecase(), deps),
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

	mqttbroker "github.com/golangid/candi-plugin/mqtt-broker"
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
	err := uc.deps.GetBroker(mqttbroker.MQTTBroker).GetPublisher().PublishMessage(ctx, &candishared.PublisherArgument{
		Topic:		"example-topic",
		Message:	"hello world",
		Header:		map[string]any{
            mqtt.ConfigHeaderQOS: byte(1),
            mqtt.ConfigHeaderRetain: false,
        },
	})
	return err
}
```
