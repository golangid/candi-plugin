package p2p

import (
	"context"
	"fmt"
	"net"

	"github.com/golangid/candi/tracer"
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

type tcpContextImpl struct {
	ctx        context.Context
	conn       net.Conn
	bufferSize int
	message    []byte
}

// Context method
func (c *tcpContextImpl) Context() context.Context {
	return c.ctx
}

// GetMessage method
func (c *tcpContextImpl) GetMessage() []byte {
	return c.message
}

// WriteResponse method
func (c *tcpContextImpl) Write(message []byte) (n int, err error) {
	messageSize := len(message)
	tracer.Log(c.ctx, "response_size", fmt.Sprintf("%d bytes", messageSize))
	tracer.Log(c.ctx, "response_message", message)

	for i := 0; i < messageSize; i += c.bufferSize {
		lastOffset := i + c.bufferSize
		if lastOffset > messageSize {
			lastOffset = messageSize
		}
		n, err := c.conn.Write(message[i:lastOffset])
		if err != nil {
			return n, err
		}
	}

	if err := c.conn.(interface{ CloseWrite() error }).CloseWrite(); err != nil {
		return n, err
	}
	return len(message), nil
}
