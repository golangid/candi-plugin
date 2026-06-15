package amazonsqs

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/logger"
)

// BrokerOptionFunc func type
type BrokerOptionFunc func(*Broker)

// BrokerSetClient set amazonsqs connection
func BrokerSetClient(client *sqs.Client) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.Client = client
	}
}

// BrokerSetPublisher set custom publisher
func BrokerSetPublisher(pub interfaces.Publisher) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.publisher = pub
	}
}

// InitDefaultConnection amazonsqs
func InitDefaultConnection(options ...func(*config.LoadOptions) error) *sqs.Client {
	// Load AWS configuration with the provided options
	cfg, err := config.LoadDefaultConfig(context.TODO(), options...)
	if err != nil {
		panic(err)
	}

	return sqs.NewFromConfig(cfg)
}

// NewAmazonSQSBroker setup AmazonSQS broker for publisher or consumer
func NewAmazonSQSBroker(client *sqs.Client, opts ...BrokerOptionFunc) *Broker {
	deferFunc := logger.LogWithDefer("Load Amazon SQS broker configuration... ")
	defer deferFunc()

	amazonSQSBroker := &Broker{
		WorkerType: AmazonSQSBroker,
		Client:     client,
	}
	for _, opt := range opts {
		opt(amazonSQSBroker)
	}

	if amazonSQSBroker.publisher == nil {
		amazonSQSBroker.publisher = NewPublisher(amazonSQSBroker.Client)
	}

	return amazonSQSBroker
}

// Broker amazonsqs
type Broker struct {
	WorkerType    types.Worker
	Client        *sqs.Client
	publisher     interfaces.Publisher
	queueURLCache sync.Map
}

// GetPublisher method
func (s *Broker) GetPublisher() interfaces.Publisher {
	return s.publisher
}

// resolveQueueURL resolve queue name (or url) into queue url, with cache
func (s *Broker) resolveQueueURL(ctx context.Context, topic string) (string, error) {
	return resolveQueueURL(ctx, s.Client, &s.queueURLCache, topic)
}

// GetName method
func (s *Broker) GetName() types.Worker {
	return AmazonSQSBroker
}

// Health method
func (s *Broker) Health() map[string]error {

	// TODO: add health check from client connection
	var err error
	return map[string]error{
		string(AmazonSQSBroker): err,
	}
}

// Disconnect method
func (s *Broker) Disconnect(ctx context.Context) error {
	deferFunc := logger.LogWithDefer("amazonsqs broker: disconnect...")
	defer deferFunc()

	return nil
}
