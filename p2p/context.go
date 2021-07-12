package p2p

import (
	"context"
	"fmt"
	"net"

	"pkg.agungdp.dev/candi/tracer"
)

type udpContextImpl struct {
	ctx        context.Context
	conn       *net.UDPConn
	clientAddr *net.UDPAddr
	message    []byte
}

// Context method
func (c *udpContextImpl) Context() context.Context {
	return c.ctx
}

// GetMessage method
func (c *udpContextImpl) GetMessage() []byte {
	return c.message
}

// WriteResponse method
func (c *udpContextImpl) Write(message []byte) (n int, err error) {
	tracer.Log(c.ctx, "response_size", fmt.Sprintf("%d bytes", len(message)))
	tracer.Log(c.ctx, "response_message", message)
	return c.conn.WriteToUDP(message, c.clientAddr)
}
