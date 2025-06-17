package mqttbroker

import (
	"context"
	"errors"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golangid/candi/codebase/factory/types"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/logger"
)

// BrokerOptionFunc func type
type BrokerOptionFunc func(*Broker)

// BrokerSetWorkerType set worker type
func BrokerSetWorkerType(workerType types.Worker) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.WorkerType = workerType
	}
}

// BrokerSetPublisher set custom publisher
func BrokerSetPublisher(pub interfaces.Publisher) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.publisher = pub
	}
}

// BrokerSetSubscriberQOS set qos value for subscriber
func BrokerSetSubscriberQOS(qos byte) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.subscriberQOS = qos
	}
}

// BrokerSetPublisherQOS set qos value for publisher
func BrokerSetPublisherQOS(qos byte) BrokerOptionFunc {
	return func(bk *Broker) {
		bk.publisherQOS = qos
	}
}

// BrokerSetMessageRetain set retain to true
func BrokerSetMessageRetain() BrokerOptionFunc {
	return func(bk *Broker) {
		bk.retain = true
	}
}

// Broker MQTT broker
type Broker struct {
	WorkerType types.Worker

	client        mqtt.Client
	publisher     interfaces.Publisher
	subscriberQOS byte
	publisherQOS  byte
	retain        bool
}

// NewMQTTBroker setup mqtt broker for publisher or consumer
func NewMQTTBroker(clientOpts *mqtt.ClientOptions, opts ...BrokerOptionFunc) *Broker {
	deferFunc := logger.LogWithDefer("Load MQTT broker configuration... ")
	defer deferFunc()

	bk := &Broker{
		WorkerType:    MQTTBroker,
		client:        mqtt.NewClient(clientOpts),
		subscriberQOS: 2,
		publisherQOS:  2,
		retain:        false,
	}
	for _, opt := range opts {
		opt(bk)
	}

	token := bk.client.Connect()
	<-token.Done()
	if err := token.Error(); err != nil {
		panic(err)
	}
	if bk.publisher == nil {
		bk.publisher = NewPublisher(bk.client, bk.publisherQOS, bk.retain)
	}

	return bk
}

// GetPublisher method
func (b *Broker) GetPublisher() interfaces.Publisher {
	return b.publisher
}

// GetName method
func (b *Broker) GetName() types.Worker {
	return MQTTBroker
}

// Health method
func (b *Broker) Health() map[string]error {
	var err error
	if !b.client.IsConnected() {
		err = errors.New("mqtt broker has been disconnected")
	}
	return map[string]error{
		string(b.WorkerType): err,
	}
}

// Disconnect method
func (b *Broker) Disconnect(ctx context.Context) error {
	defer logger.LogWithDefer("\x1b[33;5mmqtt_broker\x1b[0m: disconnect...")()

	b.client.Disconnect(500)
	return nil
}
