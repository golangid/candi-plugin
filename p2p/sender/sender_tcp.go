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
	bufferResponse := &bytes.Buffer{}
	buff := make([]byte, t.bufferSize)
	for {
		n, err := t.conn.Read(buff)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		bufferResponse.Write(buff[0:n])
	}

	return bufferResponse.Bytes(), nil
}

func (t *tcpSenderImpl) Close() error {
	return t.conn.Close()
}
