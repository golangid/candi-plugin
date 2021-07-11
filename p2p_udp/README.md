# Peer to Peer UDP Server

## Install this plugin in your `candi` service

### Add in service.go

```go
package service

import (
	p2pudp "github.com/agungdwiprasetyo/candi-plugin/p2p_udp"
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
		p2pudp.NewP2PUDP(s, "[UDP port]"),
	}...)

    ...
}
...
```

### Register in module.go

```go
package examplemodule

import (
	"example.service/internal/modules/examplemodule/delivery/p2pudphandler"
	"example.service/pkg/shared/usecase"

	p2pudp "github.com/agungdwiprasetyo/candi-plugin/p2p_udp"

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
			p2pudp.P2PUDP: p2pudphandler.NewHandler(usecase.GetSharedUsecase(), dependency.GetMiddleware(), dependency.GetValidator()),
		},
	}
}

// ...another method
```

### Create delivery handler

```go
package p2pudphandler

import (
	"encoding/json"

	"example.service/pkg/shared/usecase"

	p2pudp "github.com/agungdwiprasetyo/candi-plugin/p2p_udp"
	"pkg.agungdp.dev/candi/codebase/interfaces"
	"pkg.agungdp.dev/candi/tracer"
)

// Handler type
type Handler struct {
	mw        interfaces.Middleware
	uc        usecase.Usecase
	validator interfaces.Validator
}

// NewHandler constructor
func NewHandler(uc usecase.Usecase, mw interfaces.Middleware, validator interfaces.Validator) *Handler {
	return &Handler{
		uc:        uc,
		mw:        mw,
		validator: validator,
	}
}

// MountHandlers mount handler group
func (h *Handler) MountHandlers(i interface{}) {
	group := p2pudp.ParseGroupHandler(i)

	group.Register("test", h.handleTest)
}

func (h *Handler) handleTest(ctx p2pudp.Context) error {
	trace := tracer.StartTrace(ctx.Context(), "P2PHandler:Test")
	defer trace.Finish()

	var message = map[string]string{
		"message": "ok",
		"request": string(ctx.GetMessage()),
	}

	return json.NewEncoder(ctx).Encode(message)
}
```
