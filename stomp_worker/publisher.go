package stompworker

import (
	"context"

	"github.com/go-stomp/stomp/v3"
	"pkg.agungdp.dev/candi/candihelper"
	"pkg.agungdp.dev/candi/candishared"
	"pkg.agungdp.dev/candi/codebase/interfaces"
	"pkg.agungdp.dev/candi/tracer"
)

const (
	// StompContentTypeKey for context key
	StompContentTypeKey = candishared.ContextKey("stompContentType")
)

// publisher instance
type publisher struct {
	conn *stomp.Conn
}

// NewStompPublisher constructor
func NewStompPublisher(conn *stomp.Conn) interfaces.Publisher {
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
	trace.SetTag("content-type", contentType)
	trace.SetTag("topic", args.Topic)
	trace.SetTag("key", args.Key)
	trace.Log("data", args.Data)

	return s.conn.Send(args.Topic, contentType, candihelper.ToBytes(args.Data))
}
