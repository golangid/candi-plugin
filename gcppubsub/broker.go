package gcppubsub

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/logger"
	"google.golang.org/api/option"
)

// BrokerOptionFunc func type
type BrokerOptionFunc func(*Broker)

// BrokerSetWorkerType set worker type
func BrokerSetWorkerType(workerType types.Worker) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.WorkerType = workerType
	}
}

// BrokerSetClient set gcp client
func BrokerSetClient(client *pubsub.Client) BrokerOptionFunc {
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

// InitDefaultClient setup gcp pubsub client
func InitDefaultClient(gcpProjectName, credentialPath string) *pubsub.Client {
	client, err := pubsub.NewClient(context.Background(), gcpProjectName, option.WithCredentialsFile(credentialPath))
	if err != nil {
		panic(err)
	}

	return client
}

// Broker gcp pubsub broker
type Broker struct {
	WorkerType types.Worker
	Client     *pubsub.Client
	publisher  interfaces.Publisher
}

// NewGCPPubSubBroker setup gcp pubsub broker for publisher or consumer
func NewGCPPubSubBroker(opts ...BrokerOptionFunc) *Broker {
	deferFunc := logger.LogWithDefer("Load GCP PubSub broker configuration... ")
	defer deferFunc()

	gcpPubSubBroker := &Broker{
		WorkerType: GoogleCloudPubSub,
	}
	for _, opt := range opts {
		opt(gcpPubSubBroker)
	}

	if gcpPubSubBroker.publisher == nil {
		gcpPubSubBroker.publisher = NewPublisher(gcpPubSubBroker.Client)
	}

	return gcpPubSubBroker
}

// GetPublisher method
func (g *Broker) GetPublisher() interfaces.Publisher {
	return g.publisher
}

// GetName method
func (g *Broker) GetName() types.Worker {
	return GoogleCloudPubSub
}

// Health method
func (g *Broker) Health() map[string]error {

	// TODO: add health check from client connection
	var err error
	return map[string]error{
		string(GoogleCloudPubSub): err,
	}
}

// Disconnect method
func (g *Broker) Disconnect(ctx context.Context) error {
	deferFunc := logger.LogWithDefer("gcp pubsub: disconnect...")
	defer deferFunc()

	return g.Client.Close()
}
