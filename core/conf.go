package core

import (
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/logging"
)

// Queue2NATS type for an mq queue to nats connector
const Queue2NATS = "Queue2NATS"

// Queue2Stan type for an mq queue to stan connector
const Queue2Stan = "Queue2Stan"

// Stan2Queue type for an stan to mq queue connector
const Stan2Queue = "Stan2Queue"

// NATS2Queue type for an nats to mq queue connector
const NATS2Queue = "NATS2Queue"

// Topic2NATS type for an mq topic to nats connector
const Topic2NATS = "Topic2NATS"

// Topic2Stan type for an mq topic to stan connector
const Topic2Stan = "Topic2Stan"

// Stan2Topic type for an stan to mq topic connector
const Stan2Topic = "Stan2Topic"

// NATS2Topic type for an nats to mq topic connector
const NATS2Topic = "NATS2Topic"

// BridgeConfig holds the server configuration
type BridgeConfig struct {
	ReconnectInterval int // milliseconds

	NATS NATSConnectionConfig
	STAN NATSStreamingConnectionConfig

	Logging logging.Config

	Connect []ConnectionConfig
}

// DefaultBridgeConfig generates a default configuration with
// logging set to colors, time, debug and trace
// reconnect interval set to 5000 ms (5s)
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		ReconnectInterval: 5000,
		Logging: logging.Config{
			Colors: true,
			Time:   true,
			Debug:  true,
			Trace:  true,
		},
		STAN: NATSStreamingConnectionConfig{
			PubAckWait:         5000,
			DiscoverPrefix:     stan.DefaultDiscoverPrefix,
			MaxPubAcksInflight: stan.DefaultMaxPubAcksInflight,
			ConnectWait:        2000,
		},
	}
}

// TLSConf holds the configuration for a TLS connection/server
type TLSConf struct {
	Key  string
	Cert string
	Root string
}

// MQConnectionConfig configuration for an MQ Connection
type MQConnectionConfig struct {
	ConnectionName string
	ChannelName    string
	QueueManager   string
	UserName       string
	Password       string
}

// NATSConnectionConfig configuration for a NATS connection
type NATSConnectionConfig struct {
	Servers []string

	ConnectTimeout int //milliseconds
	ReconnectWait  int //milliseconds
	MaxReconnects  int

	TLS      TLSConf
	Username string
	Password string
}

// NATSStreamingConnectionConfig configuration for a STAN connection
type NATSStreamingConnectionConfig struct {
	ClusterID string
	ClientID  string

	PubAckWait         int //milliseconds
	DiscoverPrefix     string
	MaxPubAcksInflight int
	ConnectWait        int // milliseconds
}

// ConnectionConfig configuration for a bridge connection (of any type)
type ConnectionConfig struct {
	Type string // Can be Queue2NATS or any of the other constants

	Channel string // used for stan connections
	Subject string // Used for nats connections

	MQ    MQConnectionConfig // connection information, nats connections are shared
	Topic string             // Used for the mq side of things
	Queue string

	ExcludeHeaders bool //exclude headers, and just send the body to/from nats messages
}
