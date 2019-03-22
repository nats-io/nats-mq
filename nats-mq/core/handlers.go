package core

import (
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
)

func (bridge *BridgeServer) natsError(nc *nats.Conn, sub *nats.Subscription, err error) {
	bridge.logger.Warnf("nats error %s", err.Error())
}

func (bridge *BridgeServer) stanConnectionLost(sc stan.Conn, err error) {
	if !bridge.checkRunning() {
		return
	}
	bridge.logger.Warnf("nats streaming disconnected")

	bridge.natsLock.Lock()
	bridge.stan = nil // we lost stan
	bridge.natsLock.Unlock()

	bridge.checkConnections()
}

func (bridge *BridgeServer) natsDisconnected(nc *nats.Conn) {
	if !bridge.checkRunning() {
		return
	}
	bridge.logger.Warnf("nats disconnected")
	bridge.checkConnections()
}

func (bridge *BridgeServer) natsReconnected(nc *nats.Conn) {
	bridge.logger.Warnf("nats reconnected")
}

func (bridge *BridgeServer) natsClosed(nc *nats.Conn) {
	if bridge.checkRunning() {
		bridge.logger.Errorf("nats connection closed, shutting down bridge")
		go bridge.Stop()
	}
}

func (bridge *BridgeServer) natsDiscoveredServers(nc *nats.Conn) {
	bridge.logger.Debugf("discovered servers: %v\n", nc.DiscoveredServers())
	bridge.logger.Debugf("known servers: %v\n", nc.Servers())
}
