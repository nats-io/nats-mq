package core

import (
	"fmt"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// NATS2TopicConnector connects a NATS subject to an MQ topic
type NATS2TopicConnector struct {
	BridgeConnector

	topic *ibmmq.MQObject
	sub   *nats.Subscription
}

// NewNATS2TopicConnector create a nats to MQ connector
func NewNATS2TopicConnector(bridge *BridgeServer, config conf.ConnectorConfig) Connector {
	connector := &NATS2TopicConnector{}
	connector.init(bridge, config, fmt.Sprintf("NATS:%s to Topic:%s", config.Subject, config.Topic))
	return connector
}

// Start the connector
func (mq *NATS2TopicConnector) Start() error {
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
	topicObject, err := mq.connectToTopic(mq.config.Topic)

	if err != nil {
		return err
	}

	mq.topic = topicObject

	sub, err := mq.subscribeToNATS(mq.config.Subject, mq.topic)
	if err != nil {
		return err
	}
	mq.sub = sub

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and reading %s", mq.config.Topic)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

// Shutdown the connector
func (mq *NATS2TopicConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.AddDisconnect()

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	if mq.sub != nil {
		mq.sub.Unsubscribe()
		mq.sub = nil
	}

	var err error

	topic := mq.topic
	mq.topic = nil

	if topic != nil {
		err = topic.Close(0)
	}

	if mq.qMgr != nil {
		_ = mq.qMgr.Disc()
		mq.qMgr = nil
		mq.bridge.Logger().Tracef("disconnected from queue manager for %s", mq.String())
	}

	return err // ignore the disconnect error
}

// CheckConnections ensures the nats/stan connection and report an error if it is down
func (mq *NATS2TopicConnector) CheckConnections() error {
	if !mq.bridge.CheckNATS() {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}
	return nil
}
