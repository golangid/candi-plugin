package amazonsqs

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/tracer"
	"github.com/google/uuid"
)

const (
	// AmazonSQSContentTypeKey for context key
	AmazonSQSContentTypeKey = candishared.ContextKey("amazonSQSContentType")
	// AmazonSQS for event id
	AmazonSQSEventID = "amazonSQSEventID"
)

// publisher instance
type publisher struct {
	client *sqs.Client
}

// NewPublisher constructor
func NewPublisher(client *sqs.Client) interfaces.Publisher {
	return &publisher{
		client: client,
	}
}

// PublishMessage method
func (p *publisher) PublishMessage(ctx context.Context, args *candishared.PublisherArgument) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "AmazonSQSPublisher:PublishMessage")
	defer trace.Finish()

	contentType, ok := candishared.GetValueFromContext(ctx, AmazonSQSContentTypeKey).(string)
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
		AmazonSQSEventID: uuid.NewString(),
	}
	trace.InjectRequestHeader(header)

	result, err := p.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(message)),
		QueueUrl:    &args.Topic,
	})
	if err == nil {
		trace.SetTag("message_id", result.MessageId)
		trace.SetTag("sequence_number", result.SequenceNumber)
	}
	return err
}
