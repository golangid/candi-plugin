# Fiber REST Server

https://github.com/gofiber/fiber

## Install this plugin in your `candi` service

```sh
$ go get github.com/golangid/candi-plugin/fiber_rest
```

### Add in service.go

```go
package service

import (
    fiberrest "github.com/golangid/candi-plugin/fiber_rest"
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
		fiberrest.NewFiberServer(s, "[http port]", fiberrest.JaegerTracingMiddleware),,
	}...)

    ...
}
...
```

### Register in module.go

```go
package examplemodule

import (
	"example.service/internal/modules/examplemodule/delivery/fiberresthandler"
	"example.service/pkg/shared/usecase"

	fiberrest "github.com/golangid/candi-plugin/fiber_rest"

	"github.com/golangid/candi/codebase/factory/dependency"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
)

type Module struct {
	// ...another delivery handler
	serverHandlers map[types.Server]interfaces.ServerHandler
}

func NewModules(deps dependency.Dependency) *Module {
	return &Module{
		serverHandlers: map[types.Server]interfaces.ServerHandler{
			// ...another server handler
			// ...
			fiberrest.FiberREST: fiberresthandler.NewFiberHandler(usecase.GetSharedUsecase(), dependency.GetMiddleware(), dependency.GetValidator()),
		},
	}
}

// ...another method
```

### Create delivery handler

```go
package fiberresthandler

import (
	"context"
	"encoding/json"

	"example.service/pkg/shared/usecase"

	fiberrest "github.com/golangid/candi-plugin/fiber_rest"
	"github.com/gofiber/fiber/v2"

	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/tracer"
)

// FiberHandler struct
type FiberHandler struct {
	mw        interfaces.Middleware
	uc        usecase.UserUsecase
	validator interfaces.Validator
}

// NewFiberHandler constructor
func NewFiberHandler(uc usecase.UserUsecase, mw interfaces.Middleware, validator interfaces.Validator) *FiberHandler {
	return &FiberHandler{
		uc:        uc,
		mw:        mw,
		validator: validator,
	}
}

// MountHandlers mount handler group
func (h *FiberHandler) MountHandlers(i interface{}) {
	group := fiberrest.ParseGroupHandler(i)

	group.Get("/hello", fiberrest.WrapFiberHTTPMiddleware(h.mw.HTTPBearerAuth), helloHandler)
}

func helloHandler(c *fiber.Ctx) error {
	trace, ctx := tracer.StartTraceWithContext(fiberrest.FastHTTPParseContext(c.Context()), "DeliveryHandler")
	defer trace.Finish()

	claim := candishared.ParseTokenClaimFromContext(ctx)
	log.Println(claim)
	return c.JSON(fiber.Map{"message": "Hello world!"})
}
```
