package sender

import "context"

// Sender abstract interface
type Sender interface {
	Send(ctx context.Context, handler string, message []byte) (response []byte, err error)
	Close() error
}
