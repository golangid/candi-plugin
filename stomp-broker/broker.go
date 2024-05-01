package stompbroker

import (
	"context"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/logger"
)

// BrokerOptionFunc func type
type BrokerOptionFunc func(*Broker)

// BrokerSetConn set stomp connection
func BrokerSetConn(conn *stomp.Conn) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.Conn = conn
	}
}

// BrokerSetPublisher set custom publisher
func BrokerSetPublisher(pub interfaces.Publisher) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.publisher = pub
	}
}

// InitDefaultConnection stomp
func InitDefaultConnection(broker, username, password string) *stomp.Conn {
	conn, err := stomp.Dial("tcp", broker,
		stomp.ConnOpt.Login(username, password),
		stomp.ConnOpt.Host("/"),
		stomp.ConnOpt.HeartBeatError(360*time.Second),
		stomp.ConnOpt.HeartBeatGracePeriodMultiplier(3),
	)
	if err != nil {
		panic("STOMP: cannot connect to server broker: " + err.Error())
	}
	return conn
}

// NewSTOMPBroker setup STOMP broker for publisher or consumer
func NewSTOMPBroker(conn *stomp.Conn, opts ...BrokerOptionFunc) *Broker {
	deferFunc := logger.LogWithDefer("Load STOMP broker configuration... ")
	defer deferFunc()

	stompBroker := &Broker{
		WorkerType: STOMPBroker,
		Conn:       conn,
	}
	for _, opt := range opts {
		opt(stompBroker)
	}

	if stompBroker.publisher == nil {
		stompBroker.publisher = NewPublisher(stompBroker.Conn)
	}

	return stompBroker
}

// Broker stomp
type Broker struct {
	WorkerType types.Worker
	Conn       *stomp.Conn
	publisher  interfaces.Publisher
}

// GetPublisher method
func (s *Broker) GetPublisher() interfaces.Publisher {
	return s.publisher
}

// GetName method
func (s *Broker) GetName() types.Worker {
	return STOMPBroker
}

// Health method
func (s *Broker) Health() map[string]error {

	// TODO: add health check from client connection
	var err error
	return map[string]error{
		string(STOMPBroker): err,
	}
}

// Disconnect method
func (s *Broker) Disconnect(ctx context.Context) error {
	deferFunc := logger.LogWithDefer("stomp broker: disconnect...")
	defer deferFunc()

	return nil
}
