package core

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nats-mq/nats-mq/logging"
)

var version = "0.0.0-dev"

// BridgeServer represents the bridge server
type BridgeServer struct {
	sync.Mutex
	running bool

	startTime time.Time
	config    conf.BridgeConfig
	logger    logging.Logger

	nats *nats.Conn
	stan stan.Conn

	connectors  []Connector
	replyToInfo map[string]conf.ConnectorConfig

	httpReqStats     map[string]int64
	httpHandler      *http.ServeMux
	monitoringServer *http.Server
	httpListener     net.Listener
	monitoringURL    string
}

// NewBridgeServer creates a new bridge server with a default logger
func NewBridgeServer() *BridgeServer {
	return &BridgeServer{
		logger: logging.NewNATSLogger(logging.Config{
			Colors: true,
			Time:   true,
			Debug:  true,
			Trace:  true,
		}),
	}
}

// LoadConfigFile initialize the server's configuration from a file
func (bridge *BridgeServer) LoadConfigFile(configFile string) error {
	config := conf.DefaultBridgeConfig()

	if configFile == "" {
		configFile = os.Getenv("MQNATS_BRIDGE_CONFIG")
		if configFile != "" {
			bridge.logger.Noticef("using config specified in $MQNATS_BRIDGE_CONFIG %q", configFile)
		}
	} else {
		bridge.logger.Noticef("loading configuration from %q", configFile)
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

// LoadConfig initialize the server's configuration to an existing config object, useful for tests
// Does not initialize the config at all, use DefaultBridgeConfig() to create a default config
func (bridge *BridgeServer) LoadConfig(config conf.BridgeConfig) error {
	bridge.config = config
	return nil
}

// Start the server, will lock the server, assumes the config is loaded
func (bridge *BridgeServer) Start() error {
	bridge.Lock()
	defer bridge.Unlock()

	if bridge.logger != nil {
		bridge.logger.Close()
	}

	bridge.running = true
	bridge.startTime = time.Now()
	bridge.logger = logging.NewNATSLogger(bridge.config.Logging)
	bridge.replyToInfo = map[string]conf.ConnectorConfig{}
	bridge.connectors = []Connector{}

	bridge.logger.Noticef("starting MQ-NATS bridge, version %s", version)
	bridge.logger.Noticef("server time is %s", bridge.startTime.Format(time.UnixDate))

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

	if err := bridge.startMonitoring(); err != nil {
		return err
	}

	return nil
}

// Stop the bridge server
func (bridge *BridgeServer) Stop() {
	bridge.Lock()
	defer bridge.Unlock()

	bridge.logger.Noticef("stopping bridge")

	bridge.running = false

	for _, c := range bridge.connectors {
		err := c.Shutdown()

		if err != nil {
			bridge.logger.Noticef("error shutting down connector %s", err.Error())
		}
	}

	if bridge.nats != nil {
		bridge.nats.Close()
		bridge.logger.Noticef("disconnected from NATS")
	}

	if bridge.stan != nil {
		bridge.stan.Close()
		bridge.logger.Noticef("disconnected from NATS streaming")
	}

	err := bridge.StopMonitoring()
	if err != nil {
		bridge.logger.Noticef("error shutting down monitoring server %s", err.Error())
	}
}

func (bridge *BridgeServer) initializeConnectors() error {
	connectorConfigs := bridge.config.Connect

	for _, c := range connectorConfigs {
		connector, err := CreateConnector(c, bridge)

		if err != nil {
			return err
		}

		bridge.connectors = append(bridge.connectors, connector)
	}
	return nil
}

func (bridge *BridgeServer) startConnectors() error {
	for _, c := range bridge.connectors {
		if err := c.Start(); err != nil {
			bridge.logger.Noticef("error starting %s, %s", c.String(), err.Error())
			return err
		}
	}
	return nil
}

// ConnectorError is called by a connector if it has a failure that requires a reconnect
func (bridge *BridgeServer) ConnectorError(connector Connector, err error) {
	description := connector.String()
	bridge.logger.Errorf("a connector error has occurred, trying to restart %s, %s", description, err.Error())

	err = connector.Shutdown()

	if err != nil {
		bridge.logger.Noticef("error shutting down connector %s, trying to restart, %s", description, err.Error())
	}

	bridge.restartConnector(connector, description)
}

func (bridge *BridgeServer) restartConnector(connector Connector, description string) {
	bridge.Lock()
	defer bridge.Unlock()

	if !bridge.running {
		return
	}

	err := connector.Start()

	if err != nil {
		interval := bridge.config.ReconnectInterval
		bridge.logger.Noticef("error restarting connector %s, will retry in %d milliseconds, %s", description, interval, err.Error())
		timer := time.NewTimer(time.Duration(interval) * time.Millisecond)
		go func() {
			<-timer.C
			bridge.logger.Noticef("trying to restart connector %s", description)
			bridge.restartConnector(connector, description)
		}()
	}
}

// NATS hosts a shared nats connection for the connectors
func (bridge *BridgeServer) NATS() *nats.Conn {
	return bridge.nats
}

// Stan hosts a shared streaming connection for the connectors
func (bridge *BridgeServer) Stan() stan.Conn {
	return bridge.stan
}

// Logger hosts a shared logger
func (bridge *BridgeServer) Logger() logging.Logger {
	return bridge.logger
}

// RegisterReplyInfo tracks incoming descriptions so that reply to values can be mapped correctly
func (bridge *BridgeServer) RegisterReplyInfo(desc string, config conf.ConnectorConfig) {
	bridge.replyToInfo[desc] = config
}

func (bridge *BridgeServer) connectToNATS() error {
	bridge.logger.Noticef("connecting to NATS core")

	config := bridge.config.NATS
	options := []nats.Option{nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(time.Duration(config.ReconnectWait) * time.Millisecond),
		nats.Timeout(time.Duration(config.ConnectTimeout) * time.Millisecond),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			bridge.logger.Errorf("nats error %s", err.Error())
		}),
		nats.DiscoveredServersHandler(func(nc *nats.Conn) {
			bridge.logger.Debugf("discovered servers: %v\n", nc.DiscoveredServers())
			bridge.logger.Debugf("known servers: %v\n", nc.Servers())
		}),
		nats.DisconnectHandler(func(nc *nats.Conn) {
			if !bridge.running { // skip the lock, worst case we print something extra
				return
			}
			bridge.logger.Debugf("nats connection disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			bridge.logger.Debugf("nats connection reconnected")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			if bridge.running {
				bridge.logger.Errorf("nats connection closed, shutting down bridge")
				bridge.Lock()
				go bridge.Stop()
				bridge.Unlock()
			}
		}),
	}

	if config.TLS.Root != "" {
		options = append(options, nats.RootCAs(config.TLS.Root))
	}

	if config.TLS.Cert != "" {
		options = append(options, nats.ClientCert(config.TLS.Cert, config.TLS.Key))
	}

	nc, err := nats.Connect(strings.Join(config.Servers, ","),
		options...,
	)

	if err != nil {
		return err
	}

	bridge.nats = nc
	return nil
}

func (bridge *BridgeServer) connectToSTAN() error {

	if bridge.config.STAN.ClusterID == "" {
		bridge.logger.Noticef("skipping NATS streaming connection, not configured")
		return nil
	}

	bridge.logger.Noticef("connecting to NATS streaming")
	config := bridge.config.STAN

	sc, err := stan.Connect(config.ClusterID, config.ClientID,
		stan.NatsConn(bridge.nats),
		stan.PubAckWait(time.Duration(config.PubAckWait)*time.Millisecond),
		stan.MaxPubAcksInflight(config.MaxPubAcksInflight),
		stan.ConnectWait(time.Duration(config.ConnectWait)*time.Millisecond),
		func(o *stan.Options) error {
			o.DiscoverPrefix = config.DiscoverPrefix
			return nil
		})
	if err != nil {
		return err
	}
	bridge.stan = sc

	return nil
}
