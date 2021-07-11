package p2pudp

import (
	"context"
	"net"

	"pkg.agungdp.dev/candi/codebase/factory/types"
)

const (
	// P2PUDP types
	P2PUDP types.Server = "p2p_udp"

	// EOF const
	EOF = "EOF"
)

// HandlerFunc types
type HandlerFunc func(c Context) error

// HandlerGroup types
type HandlerGroup struct {
	Handlers []struct {
		Prefix      string
		HandlerFunc HandlerFunc
	}
}

// Register method from HandlerGroup
func (h *HandlerGroup) Register(prefix string, handlerFunc HandlerFunc) {
	h.Handlers = append(h.Handlers, struct {
		Prefix      string
		HandlerFunc HandlerFunc
	}{
		Prefix: prefix, HandlerFunc: handlerFunc,
	})
}

type (
	// Context type
	Context interface {
		Context() context.Context
		GetMessage() []byte
		Write(message []byte) (n int, err error)
	}

	contextImpl struct {
		ctx        context.Context
		conn       *net.UDPConn
		clientAddr *net.UDPAddr
		message    []byte
	}
)

// Context method
func (c *contextImpl) Context() context.Context {
	return c.ctx
}

// GetMessage method
func (c *contextImpl) GetMessage() []byte {
	return c.message
}

// WriteResponse method
func (c *contextImpl) Write(message []byte) (n int, err error) {
	return c.conn.WriteToUDP(message, c.clientAddr)
}

// ParseGroupHandler parse mount handler param
func ParseGroupHandler(i interface{}) *HandlerGroup {
	return i.(*HandlerGroup)
}
