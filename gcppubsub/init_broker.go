package gcppubsub

import (
	"context"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
	"pkg.agungdp.dev/candi/broker"
	"pkg.agungdp.dev/candi/codebase/interfaces"
	"pkg.agungdp.dev/candi/logger"
)

// InitDefaultClient setup gcp pubsub client
func InitDefaultClient(gcpProjectName, credentialPath string) *pubsub.Client {
	client, err := pubsub.NewClient(context.Background(), gcpProjectName, option.WithCredentialsFile(credentialPath))
	if err != nil {
		panic(err)
	}

	return client
}

// SetGCPPubSubBroker setup gcp pubsub broker for publisher or consumer
func SetGCPPubSubBroker(client *pubsub.Client) broker.OptionFunc {
	deferFunc := logger.LogWithDefer("Load GCP PubSub broker configuration... ")
	defer deferFunc()

	gcpPubSubBroker := &Broker{
		client:    client,
		publisher: NewPublisher(client),
	}
	return func(bi *broker.Broker) {
		bi.RegisterBroker(GoogleCloudPubSub, gcpPubSubBroker)
	}
}

// Broker gcp pubsub broker
type Broker struct {
	client    *pubsub.Client
	publisher interfaces.Publisher
}

// GetConfiguration method
func (g *Broker) GetConfiguration() interface{} {
	return g.client
}

// GetPublisher method
func (g *Broker) GetPublisher() interfaces.Publisher {
	return g.publisher
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

	return g.client.Close()
}
