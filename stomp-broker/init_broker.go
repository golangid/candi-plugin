package stompbroker

import (
	"context"
	"time"

	"github.com/go-stomp/stomp/v3"
	"pkg.agungdp.dev/candi/broker"
	"pkg.agungdp.dev/candi/codebase/interfaces"
	"pkg.agungdp.dev/candi/logger"
)

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

// SetSTOMPBroker setup STOMP broker for publisher or consumer
func SetSTOMPBroker(conn *stomp.Conn) broker.OptionFunc {
	deferFunc := logger.LogWithDefer("Load STOMP broker configuration... ")
	defer deferFunc()

	bk := &stompBroker{
		conn:      conn,
		publisher: NewPublisher(conn),
	}
	return func(bi *broker.Broker) {
		bi.RegisterBroker(STOMPBroker, bk)
	}
}

type stompBroker struct {
	conn      *stomp.Conn
	publisher interfaces.Publisher
}

// GetConfiguration method
func (s *stompBroker) GetConfiguration() interface{} {
	return s.conn
}

// GetPublisher method
func (s *stompBroker) GetPublisher() interfaces.Publisher {
	return s.publisher
}

// Health method
func (s *stompBroker) Health() map[string]error {

	// TODO: add health check from client connection
	var err error
	return map[string]error{
		string(STOMPBroker): err,
	}
}

// Disconnect method
func (s *stompBroker) Disconnect(ctx context.Context) error {
	deferFunc := logger.LogWithDefer("stomp broker: disconnect...")
	defer deferFunc()

	return nil
}
