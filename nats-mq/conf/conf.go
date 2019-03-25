package conf

import (
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/nats-mq/logging"
)

// Queue2NATS type for an mq queue to nats connector
const Queue2NATS = "Queue2NATS"

// Queue2Stan type for an mq queue to stan connector
const Queue2Stan = "Queue2Stan"

// Stan2Queue type for a stan to mq queue connector
const Stan2Queue = "Stan2Queue"

// NATS2Queue type for a nats to mq queue connector
const NATS2Queue = "NATS2Queue"

// Topic2NATS type for an mq topic to nats connector
const Topic2NATS = "Topic2NATS"

// Topic2Stan type for an mq topic to stan connector
const Topic2Stan = "Topic2Stan"

// Stan2Topic type for a stan to mq topic connector
const Stan2Topic = "Stan2Topic"

// NATS2Topic type for a nats to mq topic connector
const NATS2Topic = "NATS2Topic"

// BridgeConfig holds the server configuration
type BridgeConfig struct {
	ReconnectInterval int // milliseconds

	NATS NATSConfig
	STAN NATSStreamingConfig

	Logging    logging.Config
	Monitoring MonitoringConfig

	Connect []ConnectorConfig
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
			Debug:  false,
			Trace:  false,
		},
		STAN: NATSStreamingConfig{
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

// MonitoringConfig is used to define the host and port for monitoring
// The HTTPPort vs HTTPSPort setting is used to determine if security is
// enabled. By default the ports are 0 and monitoring is disabled. Set
// a port to -1 to use ephemeral ports.
// Similarly the host defaults to "" which indicates all network interfaces.
type MonitoringConfig struct {
	HTTPHost  string
	HTTPPort  int
	HTTPSPort int
	TLS       TLSConf
}

// MQConfig configuration for an MQ Connection
type MQConfig struct {
	ConnectionName string
	ChannelName    string
	QueueManager   string

	UserName string
	Password string

	KeyRepository    string
	CertificateLabel string
	SSLPeerName      string
}

// NATSConfig configuration for a NATS connection
type NATSConfig struct {
	Servers []string

	ConnectTimeout int //milliseconds
	ReconnectWait  int //milliseconds
	MaxReconnects  int

	TLS      TLSConf
	Username string
	Password string
}

// NATSStreamingConfig configuration for a STAN connection
type NATSStreamingConfig struct {
	ClusterID string
	ClientID  string

	PubAckWait         int //milliseconds
	DiscoverPrefix     string
	MaxPubAcksInflight int
	ConnectWait        int // milliseconds
}

// ConnectorConfig configuration for a bridge connection (of any type)
type ConnectorConfig struct {
	ID   string // user specified id for a connector, will be defaulted if none is provided
	Type string // Can be Queue2NATS or any of the other constants

	Channel         string // Used for stan connections
	DurableName     string // Optional, used for stan connections
	StartAtSequence int64  // Start position for stan connection, -1 means StartWithLastReceived, 0 means DeliverAllAvailable (default)
	StartAtTime     int64  // Start time, as Unix, time takes precedence over sequence

	Subject string // Used for nats connections

	MQ    MQConfig // Connection information, nats connections are shared
	Topic string   // Used for the mq side of things
	Queue string

	IncomingBufferSize  int // buffer size for polling (unused currently)
	IncomingMessageWait int // wait time for polling in ms (unused currently)

	MaxMQMessagesInFlight int
	ExcludeHeaders        bool //exclude headers, and just send the body to/from nats messages
}
