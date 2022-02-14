package gcppubsub

import (
	"context"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/tracer"
)

type publisher struct {
	client *pubsub.Client
}

// NewPublisher gcp
func NewPublisher(client *pubsub.Client) interfaces.Publisher {
	return &publisher{
		client: client,
	}
}

func (p *publisher) PublishMessage(ctx context.Context, args *candishared.PublisherArgument) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "GCPPubSub:PublishMessage")
	defer trace.Finish()

	var msg []byte
	if len(args.Message) > 0 {
		msg = args.Message
	} else {
		msg = candihelper.ToBytes(args.Data)
	}

	message := &pubsub.Message{
		Data:        msg,
		PublishTime: time.Now(),
	}
	message.Attributes = make(map[string]string)
	trace.InjectRequestHeader(message.Attributes)
	for k, v := range args.Header {
		message.Attributes[k] = string(candihelper.ToBytes(v))
	}

	result := p.client.Topic(args.Topic).Publish(ctx, message)
	serverID, err := result.Get(ctx)
	trace.Log("server_id", serverID)
	return err
}
