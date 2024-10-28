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
 *
 */

package core

import (
	"fmt"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// Topic2NATSConnector connects an MQ queue to a NATS subject
type Topic2NATSConnector struct {
	BridgeConnector

	topic      *ibmmq.MQObject
	sub        *ibmmq.MQObject
	shutdownCB ShutdownCallback
}

// NewTopic2NATSConnector create a new MQ to Stan connector
func NewTopic2NATSConnector(bridge *BridgeServer, config conf.ConnectorConfig) Connector {
	connector := &Topic2NATSConnector{}
	connector.init(bridge, config, fmt.Sprintf("MQ:%s to NATS:%s", config.Topic, config.Subject))
	return connector
}

// Start the connector
func (mq *Topic2NATSConnector) Start() error {
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

	topic, sub, err := mq.subscribeToTopic(mq.config.Topic)

	if err != nil {
		return err
	}

	mq.topic = topic
	mq.sub = sub

	mq.bridge.Logger().Tracef("subscribed to %s", mq.config.Topic)

	cb, err := mq.setUpListener(mq.topic, mq.natsMessageHandler, mq)
	if err != nil {
		return err
	}
	mq.shutdownCB = cb

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and subscribed to %s", mq.config.Topic)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

// Shutdown the connector
func (mq *Topic2NATSConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.AddDisconnect()

	if mq.topic == nil {
		return nil
	}

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	sub := mq.sub
	topic := mq.topic
	mq.topic = nil
	mq.sub = nil

	if mq.shutdownCB != nil {
		if err := mq.shutdownCB(); err != nil {
			mq.bridge.Logger().Noticef("error stopping listener for %s, %s", mq.String(), err.Error())
		}
		mq.shutdownCB = nil
	}

	if sub != nil {
		if err := sub.Close(0); err != nil {
			mq.bridge.Logger().Noticef("error closing subscription for %s", mq.String())
		}
	}

	if topic != nil {
		if err := topic.Close(0); err != nil {
			mq.bridge.Logger().Noticef("error closing topic for %s", mq.String())
		}
	}

	if mq.qMgr != nil {
		mq.bridge.Logger().Noticef("shutting down qmgr")
		if err := mq.qMgr.Disc(); err != nil {
			mq.bridge.Logger().Noticef("error disconnecting from queue manager for %s, %s", mq.String(), err.Error())
		}
		mq.qMgr = nil
		mq.bridge.Logger().Tracef("disconnected from queue manager for %s", mq.String())
	}

	return nil
}

// CheckConnections ensures the nats/stan connection and report an error if it is down
func (mq *Topic2NATSConnector) CheckConnections() error {
	if !mq.bridge.CheckNATS() {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}
	return nil
}
