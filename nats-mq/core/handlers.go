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
	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
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
