package p2p

import (
	"context"

	"pkg.agungdp.dev/candi/codebase/factory/types"
)

const (
	// P2PUDP types
	P2PUDP types.Server = "p2p_udp"

	// EOF const
	EOF = "EOF"
)

type (
	// HandlerFunc types
	HandlerFunc func(c Context) error

	// HandlerGroup types
	HandlerGroup struct {
		Handlers []struct {
			Prefix      string
			HandlerFunc HandlerFunc
		}
	}

	// Context type
	Context interface {
		Context() context.Context
		GetMessage() []byte
		Write(message []byte) (n int, err error)
	}
)

// Register method from HandlerGroup
func (h *HandlerGroup) Register(prefix string, handlerFunc HandlerFunc) {
	h.Handlers = append(h.Handlers, struct {
		Prefix      string
		HandlerFunc HandlerFunc
	}{
		Prefix: prefix, HandlerFunc: handlerFunc,
	})
}

// ParseGroupHandler parse mount handler param
func ParseGroupHandler(i interface{}) *HandlerGroup {
	return i.(*HandlerGroup)
}
