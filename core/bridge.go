package core

import (
	"fmt"
	"github.com/nats-io/nats-mq/conf"
	"github.com/nats-io/nats-mq/logging"
	"os"
	"sync"
	"time"

	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
)

var version = "0.0.0-dev"

// BridgeServer represents the bridge server
type BridgeServer struct {
	sync.Mutex
	running bool

	startTime time.Time
	config    BridgeConfig
	Logger    logging.Logger

	nats *nats.Conn
	stan stan.Conn

	connectors []Connector
}

// Connector is used to type the various connectors in the bridge, primarily for shutdown
type Connector interface {
	Start() error
	Shutdown() error
	String() string
	Config() ConnectionConfig
}

// NewBridgeServer creates a new bridge server with a default logger
func NewBridgeServer() *BridgeServer {
	return &BridgeServer{
		Logger: logging.NewNATSLogger(logging.Config{
			Colors: true,
			Time:   true,
			Debug:  true,
			Trace:  true,
		}),
	}
}

// LoadConfigFile initialize the server's configuration from a file
func (bridge *BridgeServer) LoadConfigFile(configFile string) error {
	config := DefaultBridgeConfig()

	if configFile == "" {
		configFile = os.Getenv("MQNATS_BRIDGE_CONFIG")
		if configFile != "" {
			bridge.Logger.Noticef("using config specified in $MQNATS_BRIDGE_CONFIG %q", configFile)
		}
	} else {
		bridge.Logger.Noticef("loading configuration from %q", configFile)
	}

	if configFile == "" {
		return fmt.Errorf("no config file specified")
	}

	if err := conf.LoadConfigFromFile(configFile, &config, false); err != nil {
		return err
	}

	bridge.config = config
	return nil
}

// LoadConfigString initialize the server's configuration from a string, useful for tests
func (bridge *BridgeServer) LoadConfigString(configString string) error {
	config := DefaultBridgeConfig()

	if configString == "" {
		return fmt.Errorf("no config string specified")
	}

	if err := conf.LoadConfigFromString(configString, &config, false); err != nil {
		return err
	}

	bridge.config = config
	return nil
}

// LoadConfig initialize the server's configuration to an existing config object, useful for tests
// Does not initialize the config at all, use DefaultBridgeConfig() to create a default config
func (bridge *BridgeServer) LoadConfig(config BridgeConfig) error {
	bridge.config = config
	return nil
}

// Start the server, will lock the server, assumes the config is loaded
func (bridge *BridgeServer) Start() error {
	bridge.Lock()
	defer bridge.Unlock()

	if bridge.Logger != nil {
		bridge.Logger.Close()
	}

	bridge.running = true
	bridge.startTime = time.Now()
	bridge.Logger = logging.NewNATSLogger(bridge.config.Logging)

	bridge.Logger.Noticef("starting MQ-NATS bridge, version %s", version)
	bridge.Logger.Noticef("server time is %s", bridge.startTime.Format(time.UnixDate))

	if err := bridge.connectToNATS(); err != nil {
		return err
	}

	if err := bridge.connectToSTAN(); err != nil {
		return err
	}

	if err := bridge.initializeConnectors(); err != nil {
		return err
	}

	if err := bridge.startConnectors(); err != nil {
		return err
	}

	return nil
}

// Stop the bridge server
func (bridge *BridgeServer) Stop() {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.Logger.Noticef("stopping bridge...")

	bridge.running = false

	for _, c := range bridge.connectors {
		err := c.Shutdown()

		if err != nil {
			bridge.Logger.Noticef("error shutting down connector %s", err.Error())
		}
	}

	if bridge.nats != nil {
		bridge.nats.Close()
		bridge.Logger.Noticef("disconnected from NATS")
	}

	if bridge.stan != nil {
		bridge.stan.Close()
		bridge.Logger.Noticef("disconnected from NATS streaming")
	}
}

func (bridge *BridgeServer) initializeConnectors() error {
	connectorConfigs := bridge.config.Connect

	for _, c := range connectorConfigs {
		connector, err := bridge.createConnector(c)

		if err != nil {
			return err
		}

		bridge.connectors = append(bridge.connectors, connector)
	}
	return nil
}

func (bridge *BridgeServer) createConnector(config ConnectionConfig) (Connector, error) {
	switch config.Type {
	case Queue2NATS:
		return NewQueue2NATSConnector(bridge, config), nil
	case NATS2Queue:
		return NewNATS2QueueConnector(bridge, config), nil
	case Queue2Stan:
		return NewQueue2STANConnector(bridge, config), nil
	case Stan2Queue:
		return NewStan2QueueConnector(bridge, config), nil
	default:
		return nil, fmt.Errorf("unknown connector type %q in configuration", config.Type)
	}
}

func (bridge *BridgeServer) startConnectors() error {
	for _, c := range bridge.connectors {
		if err := c.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (bridge *BridgeServer) connectorError(connector Connector, err error) {
	bridge.Lock()
	description := connector.String()
	bridge.Logger.Errorf("a connector error has occurred, trying to restart %s, %s", description, err.Error())

	err = connector.Shutdown()

	if err != nil {
		bridge.Logger.Noticef("error shutting down connector %s, trying to restart, %s", description, err.Error())
	}

	count := len(bridge.connectors)
	for i, c := range bridge.connectors {
		if c == connector {
			bridge.connectors[i] = bridge.connectors[count-1]
			bridge.connectors[count-1] = nil // so we can gc it
			bridge.connectors = bridge.connectors[:count-1]
			break
		}
	}
	bridge.Unlock()

	bridge.restartConnector(connector.Config(), description)
}

func (bridge *BridgeServer) restartConnector(config ConnectionConfig, description string) {
	bridge.Lock()
	defer bridge.Unlock()

	if !bridge.running {
		return
	}

	newConnector, err := bridge.createConnector(config)

	if err != nil {
		bridge.Logger.Noticef("error creating connector %s for restart, %s", description, err.Error())
	}

	err = newConnector.Start()

	if err != nil {
		interval := bridge.config.ReconnectInterval
		bridge.Logger.Noticef("error restarting connector %s, will retry in %d milliseconds, %s", description, interval, err.Error())
		timer := time.NewTimer(time.Duration(interval) * time.Millisecond)
		go func() {
			<-timer.C
			bridge.Logger.Noticef("trying to restart connector %s", description)
			bridge.restartConnector(config, description)
		}()
	}

	bridge.connectors = append(bridge.connectors, newConnector)
}
