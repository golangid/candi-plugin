package stompbroker

import (
	"context"

	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/tracer"
	"github.com/google/uuid"
)

const (
	// StompContentTypeKey for context key
	StompContentTypeKey = candishared.ContextKey("stompContentType")
	// StompEventID for event id
	StompEventID = "stompEventID"
)

// publisher instance
type publisher struct {
	conn *stomp.Conn
}

// NewPublisher constructor
func NewPublisher(conn *stomp.Conn) interfaces.Publisher {
	return &publisher{
		conn: conn,
	}
}

// PublishMessage method
func (s *publisher) PublishMessage(ctx context.Context, args *candishared.PublisherArgument) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "StompPublisher:PublishMessage")
	defer trace.Finish()

	contentType, ok := candishared.GetValueFromContext(ctx, StompContentTypeKey).(string)
	if !ok {
		contentType = "text/plain"
	}

	var message []byte
	if len(args.Message) > 0 {
		message = args.Message
	} else {
		message = candihelper.ToBytes(args.Data)
	}

	trace.SetTag("content-type", contentType)
	trace.SetTag("topic", args.Topic)
	trace.SetTag("key", args.Key)
	trace.Log("message", message)

	header := map[string]string{
		StompEventID: uuid.NewString(),
	}
	trace.InjectRequestHeader(header)

	var opts []func(*frame.Frame) error
	for k, v := range header {
		opts = append(opts, stomp.SendOpt.Header(k, v))
	}

	return s.conn.Send(
		args.Topic, contentType, message, opts...,
	)
}
