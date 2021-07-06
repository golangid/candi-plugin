# Fiber REST Server

https://github.com/gofiber/fiber

## Install this plugin in your `candi` service

### Add in service.go

```go
package service

import (
    fiberrest "github.com/agungdwiprasetyo/candi-plugin/fiber_rest"
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
	"example.service/internal/modules/examplemodule/delivery/workerhandler"

    fiberrest "github.com/agungdwiprasetyo/candi-plugin/fiber_rest"

	"pkg.agungdp.dev/candi/codebase/factory/dependency"
	"pkg.agungdp.dev/candi/codebase/factory/types"
	"pkg.agungdp.dev/candi/codebase/interfaces"
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
			fiberrest.FiberREST: resthandler.NewFiberHandler(usecaseUOW.User(), dependency.GetMiddleware(), dependency.GetValidator()),
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

	fiberrest "github.com/agungdwiprasetyo/candi-plugin/fiber_rest"
	"github.com/gofiber/fiber/v2"

	"pkg.agungdp.dev/candi/candishared"
	"pkg.agungdp.dev/candi/codebase/interfaces"
	"pkg.agungdp.dev/candi/tracer"
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
