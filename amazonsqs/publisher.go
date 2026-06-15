package amazonsqs

import (
	"context"
	"sync"

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
	// AmazonSQSEventID for event id
	AmazonSQSEventID = "amazonSQSEventID"
)

// publisher instance
type publisher struct {
	client        *sqs.Client
	queueURLCache *sync.Map
}

// NewPublisher constructor
func NewPublisher(client *sqs.Client) interfaces.Publisher {
	return &publisher{
		client:        client,
		queueURLCache: new(sync.Map),
	}
}

// PublishMessage method, args.Topic can be an SQS queue name or a full queue url
func (p *publisher) PublishMessage(ctx context.Context, args *candishared.PublisherArgument) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "AmazonSQSPublisher:PublishMessage")
	defer func() { trace.Finish(tracer.FinishWithError(err)) }()

	var message []byte
	if len(args.Message) > 0 {
		message = args.Message
	} else {
		message = candihelper.ToBytes(args.Data)
	}

	contentType, ok := candishared.GetValueFromContext(ctx, AmazonSQSContentTypeKey).(string)
	if !ok {
		if args.ContentType != "" {
			contentType = args.ContentType
		} else {
			contentType = "text/plain"
		}
	}

	header := map[string]string{
		AmazonSQSEventID: uuid.NewString(),
		"content-type":   contentType,
	}
	for k, v := range args.Header {
		header[k] = candihelper.ToString(v)
	}
	// inject trace context so consumers can continue the distributed trace
	trace.InjectRequestHeader(header)

	trace.SetTag("topic", args.Topic)
	trace.SetTag("key", args.Key)
	trace.Log("header", header)
	trace.Log("message", message)

	queueURL, err := resolveQueueURL(ctx, p.client, p.queueURLCache, args.Topic)
	if err != nil {
		return err
	}
	trace.SetTag("queue_url", queueURL)

	input := &sqs.SendMessageInput{
		QueueUrl:          aws.String(queueURL),
		MessageBody:       aws.String(string(message)),
		MessageAttributes: messageAttributesFromHeader(header),
	}
	if args.Delay > 0 {
		delaySeconds := int32(args.Delay.Seconds())
		if delaySeconds > 900 { // SQS max delay is 900 seconds (15 minutes)
			delaySeconds = 900
		}
		input.DelaySeconds = delaySeconds
	}

	result, err := p.client.SendMessage(ctx, input)
	if err != nil {
		return err
	}

	trace.SetTag("message_id", aws.ToString(result.MessageId))
	return nil
}
