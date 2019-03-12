package examples

import (
	"github.com/nats-io/nats-mq/core"
)

// ExampleConfiguration is used for several examples
type ExampleConfiguration struct {
	MQ   core.MQConfig
	NATS core.NATSConfig
	STAN core.NATSStreamingConfig
}
