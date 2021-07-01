package stompworker

import (
	"time"

	"github.com/go-stomp/stomp/v3"
)

type stompBroker struct {
	conn *stomp.Conn
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
