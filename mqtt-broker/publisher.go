package mqttbroker

import (
	"context"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	"github.com/golangid/candi/codebase/interfaces"
	"github.com/golangid/candi/tracer"
)

type publisher struct {
	client mqtt.Client
	qos    byte
	retain bool
}

// NewPublisher for MQTT
func NewPublisher(client mqtt.Client, defaultQOS byte, retain bool) interfaces.Publisher {
	return &publisher{
		client: client, qos: defaultQOS, retain: retain,
	}
}

func (p *publisher) PublishMessage(ctx context.Context, args *candishared.PublisherArgument) (err error) {
	trace, ctx := tracer.StartTraceWithContext(ctx, "MQTTBroker:PublishMessage")
	defer trace.Finish()

	var msg []byte
	if len(args.Message) > 0 {
		msg = args.Message
	} else {
		msg = candihelper.ToBytes(args.Data)
	}

	qos, ok := args.Header[ConfigHeaderQOS].(byte)
	if !ok {
		qos = p.qos
	}
	retain, ok := args.Header[ConfigHeaderRetain].(bool)
	if !ok {
		retain = p.retain
	}

	token := p.client.Publish(args.Topic, qos, retain, msg)
	<-token.Done()
	return token.Error()
}
