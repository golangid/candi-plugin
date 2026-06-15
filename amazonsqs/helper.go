package amazonsqs

import (
	"context"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const sqsAttributeTypeString = "String"

// resolveQueueURL resolves a topic into an SQS queue URL. When the topic already
// looks like a queue URL (http/https) it is returned as-is; otherwise it is treated
// as a queue name and looked up via GetQueueUrl. Successful lookups are cached so
// both publisher and consumer can address a queue by its name.
func resolveQueueURL(ctx context.Context, client *sqs.Client, cache *sync.Map, topic string) (string, error) {
	if strings.HasPrefix(topic, "http://") || strings.HasPrefix(topic, "https://") {
		return topic, nil
	}
	if cache != nil {
		if cached, ok := cache.Load(topic); ok {
			return cached.(string), nil
		}
	}

	out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(topic),
	})
	if err != nil {
		return "", err
	}

	queueURL := aws.ToString(out.QueueUrl)
	if cache != nil {
		cache.Store(topic, queueURL)
	}
	return queueURL, nil
}

// messageAttributesFromHeader converts a string header map into SQS message
// attributes, used to propagate trace headers and metadata to consumers.
func messageAttributesFromHeader(header map[string]string) map[string]sqstypes.MessageAttributeValue {
	if len(header) == 0 {
		return nil
	}
	attrs := make(map[string]sqstypes.MessageAttributeValue, len(header))
	for k, v := range header {
		if k == "" || v == "" {
			continue
		}
		attrs[k] = sqstypes.MessageAttributeValue{
			DataType:    aws.String(sqsAttributeTypeString),
			StringValue: aws.String(v),
		}
	}
	return attrs
}

// headerFromMessageAttributes converts SQS message attributes back into a string
// header map for the consumer.
func headerFromMessageAttributes(attrs map[string]sqstypes.MessageAttributeValue) map[string]string {
	header := make(map[string]string, len(attrs))
	for k, v := range attrs {
		if v.StringValue != nil {
			header[k] = *v.StringValue
		}
	}
	return header
}
