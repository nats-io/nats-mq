/*
 * Copyright 2012-2019 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

var version = "0.5"

// BridgeServer represents the bridge server
type BridgeServer struct {
	runningLock sync.Mutex
	running     bool

	startTime time.Time
	config    conf.BridgeConfig
	logger    logging.Logger

	natsLock sync.Mutex
	nats     *nats.Conn
	stan     stan.Conn

	connectors  []Connector
	replyToInfo map[string]conf.ConnectorConfig

	reconnectLock  sync.Mutex
	reconnect      map[string]Connector
	reconnectTimer *reconnectTimer

	statsLock        sync.Mutex
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
	bridge.runningLock.Lock()
	defer bridge.runningLock.Unlock()

	if bridge.logger != nil {
		bridge.logger.Close()
	}

	bridge.running = true
	bridge.startTime = time.Now()
	bridge.logger = logging.NewNATSLogger(bridge.config.Logging)
	bridge.replyToInfo = map[string]conf.ConnectorConfig{}
	bridge.connectors = []Connector{}
	bridge.reconnect = map[string]Connector{}

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
	bridge.runningLock.Lock()
	defer bridge.runningLock.Unlock()

	if !bridge.running {
		return // already stopped
	}

	bridge.logger.Noticef("stopping bridge")

	bridge.running = false
	bridge.stopReconnectTimer()
	bridge.reconnect = map[string]Connector{} // clear the map

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

// assumes the lock is held by the caller
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

// assumes the lock is held by the caller
func (bridge *BridgeServer) startConnectors() error {
	for _, c := range bridge.connectors {
		if err := c.Start(); err != nil {
			bridge.logger.Noticef("error starting %s, %s", c.String(), err.Error())
			return err
		}
	}
	return nil
}

func (bridge *BridgeServer) checkRunning() bool {
	bridge.runningLock.Lock()
	defer bridge.runningLock.Unlock()
	return bridge.running
}

// NATS hosts a shared nats connection for the connectors
func (bridge *BridgeServer) NATS() *nats.Conn {
	bridge.natsLock.Lock()
	defer bridge.natsLock.Unlock()
	return bridge.nats
}

// Stan hosts a shared streaming connection for the connectors
func (bridge *BridgeServer) Stan() stan.Conn {
	bridge.natsLock.Lock()
	defer bridge.natsLock.Unlock()
	return bridge.stan
}

// Logger hosts a shared logger
func (bridge *BridgeServer) Logger() logging.Logger {
	return bridge.logger
}

// CheckNATS returns true if the bridge is connected to nats
func (bridge *BridgeServer) CheckNATS() bool {
	bridge.natsLock.Lock()
	defer bridge.natsLock.Unlock()

	if bridge.nats != nil {
		return bridge.nats.ConnectedUrl() != ""
	}

	return false
}

// CheckStan returns true if the bridge is connected to stan
func (bridge *BridgeServer) CheckStan() bool {
	bridge.natsLock.Lock()
	defer bridge.natsLock.Unlock()

	if bridge.nats != nil {
		ok := bridge.nats.ConnectedUrl() != ""

		if !ok {
			return false
		}
	}

	return bridge.stan != nil
}

// RegisterReplyInfo tracks incoming descriptions so that reply to values can be mapped correctly
func (bridge *BridgeServer) RegisterReplyInfo(desc string, config conf.ConnectorConfig) {
	bridge.replyToInfo[desc] = config
}

// assumes the lock is held by the caller
func (bridge *BridgeServer) connectToNATS() error {
	bridge.natsLock.Lock()
	defer bridge.natsLock.Unlock()

	bridge.logger.Noticef("connecting to NATS core")

	config := bridge.config.NATS
	options := []nats.Option{nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(time.Duration(config.ReconnectWait) * time.Millisecond),
		nats.Timeout(time.Duration(config.ConnectTimeout) * time.Millisecond),
		nats.ErrorHandler(bridge.natsError),
		nats.DiscoveredServersHandler(bridge.natsDiscoveredServers),
		nats.DisconnectHandler(bridge.natsDisconnected),
		nats.ReconnectHandler(bridge.natsReconnected),
		nats.ClosedHandler(bridge.natsClosed),
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

// assumes the lock is held by the caller
func (bridge *BridgeServer) connectToSTAN() error {
	bridge.natsLock.Lock()
	defer bridge.natsLock.Unlock()

	if bridge.stan != nil {
		return nil // already connected
	}

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
		stan.SetConnectionLostHandler(bridge.stanConnectionLost),
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

// ConnectorError is called by a connector if it has a failure that requires a reconnect
func (bridge *BridgeServer) ConnectorError(connector Connector, err error) {
	if !bridge.checkRunning() {
		return
	}

	bridge.reconnectLock.Lock()
	defer bridge.reconnectLock.Unlock()

	_, check := bridge.reconnect[connector.ID()]

	if check {
		return // we already have that connector, no need to stop or pring any messages
	}

	description := connector.String()
	bridge.logger.Errorf("a connector error has occurred, bridge will try to restart %s, %s", description, err.Error())

	err = connector.Shutdown()

	if err != nil {
		bridge.logger.Warnf("error shutting down connector %s, bridge will try to restart, %s", description, err.Error())
	}

	bridge.reconnect[connector.ID()] = connector

	bridge.ensureReconnectTimer()
}

// checkConnections loops over the connections and has them each check check their requirements
func (bridge *BridgeServer) checkConnections() {
	bridge.logger.Warnf("checking connector requirements and will restart as needed.")

	if !bridge.checkRunning() {
		return
	}

	bridge.reconnectLock.Lock()
	defer bridge.reconnectLock.Unlock()

	for _, connector := range bridge.connectors {
		_, check := bridge.reconnect[connector.ID()]

		if check {
			continue // we already have that connector, no need to stop or pring any messages
		}

		err := connector.CheckConnections()

		if err == nil {
			continue // connector is happy
		}

		description := connector.String()
		bridge.logger.Errorf("a connector error has occurred, trying to restart %s, %s", description, err.Error())

		err = connector.Shutdown()

		if err != nil {
			bridge.logger.Warnf("error shutting down connector %s, trying to restart, %s", description, err.Error())
		}

		bridge.reconnect[connector.ID()] = connector
	}

	bridge.ensureReconnectTimer()
}

// requires the reconnect lock be held by the caller
// spawns a go routine that will acquire the lock for handling reconnect tasks
func (bridge *BridgeServer) ensureReconnectTimer() {
	if bridge.reconnectTimer != nil {
		return
	}

	timer := newReconnectTimer()
	bridge.reconnectTimer = timer

	go func() {
		var doReconnect bool
		interval := bridge.config.ReconnectInterval

		doReconnect = <-timer.After(time.Duration(interval) * time.Millisecond)
		if !doReconnect {
			return
		}

		bridge.reconnectLock.Lock()
		defer bridge.reconnectLock.Unlock()

		// Wait for nats to be reconnected
		if !bridge.CheckNATS() {
			bridge.logger.Noticef("nats connection is down, will try reconnecting to streaming and restarting connectors in %d milliseconds", interval)
			bridge.reconnectTimer = nil
			bridge.ensureReconnectTimer()
		}

		// Make sure stan is up, if it should be
		if bridge.stan == nil {
			bridge.logger.Noticef("trying to reconnect to nats streaming")
			err := bridge.connectToSTAN() // this may be a no-op if bridge.stan == nil was true but is not true once we get the lock in the connect

			if err != nil {
				bridge.logger.Noticef("error restarting streaming connection, will retry in %d milliseconds", interval, err.Error())
				bridge.reconnectTimer = nil
				bridge.ensureReconnectTimer()
			}
		}

		// Do all the reconnects
		for id, connector := range bridge.reconnect {
			bridge.logger.Noticef("trying to restart connector %s", connector.String())
			err := connector.Start()

			if err != nil {
				bridge.logger.Noticef("error restarting connector %s, will retry in %d milliseconds, %s", connector.String(), interval, err.Error())
				continue
			}

			delete(bridge.reconnect, id)
		}

		bridge.reconnectTimer = nil

		if len(bridge.reconnect) > 0 {
			bridge.ensureReconnectTimer()
		}
	}()
}

// locks the reconnect lock
func (bridge *BridgeServer) stopReconnectTimer() {
	bridge.reconnectLock.Lock()
	defer bridge.reconnectLock.Unlock()

	if bridge.reconnectTimer != nil {
		bridge.reconnectTimer.Cancel()
	}

	bridge.reconnectTimer = nil
}

func (bridge *BridgeServer) checkReconnecting() bool {
	bridge.reconnectLock.Lock()
	reconnecting := len(bridge.reconnect) > 0
	bridge.reconnectLock.Unlock()
	return reconnecting
}
