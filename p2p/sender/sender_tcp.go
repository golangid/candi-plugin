package sender

import (
	"bytes"
	"context"
	"io"
	"net"
)

type tcpSenderImpl struct {
	conn       net.Conn
	bufferSize int
	buffer     *bytes.Buffer
}

// NewTCPSender init tcp sender
func NewTCPSender(targetAddress string, bufferSize int) (Sender, error) {
	conn, err := net.Dial("tcp", targetAddress)
	if err != nil {
		return nil, err
	}
	return &tcpSenderImpl{
		conn:       conn,
		bufferSize: bufferSize,
		buffer:     new(bytes.Buffer),
	}, nil
}

func (t *tcpSenderImpl) Send(ctx context.Context, handler string, message []byte) (response []byte, err error) {

	// write to receiver
	_, err = t.conn.Write(append([]byte(handler), 58))
	if err != nil {
		return nil, err
	}
	messageSize := len(message)
	for i := 0; i < messageSize; i += t.bufferSize {
		lastOffset := i + t.bufferSize
		if lastOffset > messageSize {
			lastOffset = messageSize
		}
		_, err := t.conn.Write(message[i:lastOffset])
		if err != nil {
			return nil, err
		}
	}
	if err := t.conn.(interface{ CloseWrite() error }).CloseWrite(); err != nil {
		return nil, err
	}

	// read response message from receiver
	io.Copy(t.buffer, t.conn)
	defer t.buffer.Reset()

	return t.buffer.Bytes(), nil
}

func (t *tcpSenderImpl) Close() error {
	return t.conn.Close()
}
