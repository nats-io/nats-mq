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

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// Queue2NATSConnector connects an MQ queue to a NATS subject
type Queue2NATSConnector struct {
	BridgeConnector

	queue      *ibmmq.MQObject
	shutdownCB ShutdownCallback
}

// NewQueue2NATSConnector create a new MQ to Stan connector
func NewQueue2NATSConnector(bridge *BridgeServer, config conf.ConnectorConfig) Connector {
	connector := &Queue2NATSConnector{}
	connector.init(bridge, config, fmt.Sprintf("Queue:%s to NATS:%s", config.Queue, config.Subject))
	return connector
}

// Start the connector
func (mq *Queue2NATSConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	if !mq.bridge.CheckNATS() {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}

	mq.bridge.Logger().Tracef("starting connection %s", mq.String())

	err := mq.connectToMQ()
	if err != nil {
		return err
	}

	// Create the Object Descriptor that allows us to give the queue name
	qObject, err := mq.connectToQueue(mq.config.Queue, ibmmq.MQOO_INPUT_SHARED)
	if err != nil {
		return err
	}

	mq.queue = qObject

	cb, err := mq.setUpListener(mq.queue, mq.natsMessageHandler, mq)
	if err != nil {
		return err
	}
	mq.shutdownCB = cb

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and reading %s", mq.config.Queue)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

// Shutdown the connector
func (mq *Queue2NATSConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.AddDisconnect()

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	if mq.shutdownCB != nil {
		if err := mq.shutdownCB(); err != nil {
			mq.bridge.Logger().Noticef("error stopping listener for %s, %s", mq.String(), err.Error())
		}
		mq.shutdownCB = nil
	}

	queue := mq.queue
	mq.queue = nil

	if queue != nil {
		if err := queue.Close(0); err != nil {
			mq.bridge.Logger().Noticef("error closing queue for %s, %s", mq.String(), err.Error())
		}
	}

	if mq.qMgr != nil {
		if err := mq.qMgr.Disc(); err != nil {
			mq.bridge.Logger().Noticef("error disconnecting from queue manager for %s, %s", mq.String(), err.Error())
		}
		mq.qMgr = nil
		mq.bridge.Logger().Tracef("disconnected from queue manager for %s", mq.String())
	}

	return nil // ignore the disconnect error
}

// CheckConnections ensures the nats/stan connection and report an error if it is down
func (mq *Queue2NATSConnector) CheckConnections() error {
	if !mq.bridge.CheckNATS() {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}
	return nil
}
