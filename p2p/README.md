# Peer to Peer

Simple communication with message

## Install this plugin in your `candi` service

```sh
$ go get github.com/golangid/candi-plugin/p2p
```

### Add in service.go

```go
package service

import (
	"github.com/golangid/candi-plugin/p2p"
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
		p2p.NewP2PUDP(s, "[UDP port]", p2p.ServerSetBufferSize(1024)),
	}...)

    ...
}
...
```

### Register in module.go

```go
package examplemodule

import (
	"example.service/internal/modules/examplemodule/delivery/p2phandler"
	"example.service/pkg/shared/usecase"

	"github.com/golangid/candi-plugin/p2p"

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
			p2p.P2PUDP: p2phandler.NewHandler(usecase.GetSharedUsecase(), dependency.GetMiddleware(), dependency.GetValidator()),
		},
	}
}

// ...another method
```

### Create delivery handler

```go
package p2phandler

import (
	"encoding/json"

	"example.service/pkg/shared/usecase"

	"github.com/golangid/candi-plugin/p2p"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/tracer"
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
	group := p2p.ParseGroupHandler(i)

	group.Register("test", h.handleTest)
}

func (h *Handler) handleTest(c p2p.Context) error {
	trace := tracer.StartTrace(c.Context(), "P2PHandler:Test")
	defer trace.Finish()

	var message = map[string]string{
		"message": "ok",
		"request": string(c.GetMessage()),
	}

	return json.NewEncoder(c).Encode(message)
}
```
