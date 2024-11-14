package mqttbroker

import "github.com/golangid/candi/codebase/factory/types"

const (
	// MQTTBroker types
	MQTTBroker types.Worker = "mqtt-broker"

	ConfigHeaderQOS    string = "qos"
	ConfigHeaderRetain string = "retain"
)
