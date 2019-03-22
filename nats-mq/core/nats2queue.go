package core

import (
	"fmt"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// NATS2QueueConnector connects a NATS subject to an MQ queue
type NATS2QueueConnector struct {
	BridgeConnector

	queue *ibmmq.MQObject
	ctlo  *ibmmq.MQCTLO

	sub *nats.Subscription
}

// NewNATS2QueueConnector create a nats to MQ connector
func NewNATS2QueueConnector(bridge *BridgeServer, config conf.ConnectorConfig) Connector {
	connector := &NATS2QueueConnector{}
	connector.init(bridge, config, fmt.Sprintf("NATS:%s to Queue:%s", config.Subject, config.Queue))
	return connector
}

// Start the connector
func (mq *NATS2QueueConnector) Start() error {
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
	qObject, err := mq.connectToQueue(mq.config.Queue, ibmmq.MQOO_OUTPUT)

	if err != nil {
		return err
	}

	mq.queue = qObject

	sub, err := mq.subscribeToNATS(mq.config.Subject, mq.queue)
	if err != nil {
		return err
	}
	mq.sub = sub

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and reading %s", mq.queue)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

// Shutdown the connector
func (mq *NATS2QueueConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.AddDisconnect()

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	if mq.sub != nil {
		mq.sub.Unsubscribe()
		mq.sub = nil
	}

	var err error

	queue := mq.queue
	mq.queue = nil

	if queue != nil {
		err = queue.Close(0)
	}

	if mq.qMgr != nil {
		_ = mq.qMgr.Disc()
		mq.qMgr = nil
		mq.bridge.Logger().Tracef("disconnected from queue manager for %s", mq.String())
	}

	return err // ignore the disconnect error
}

// CheckConnections ensures the nats/stan connection and report an error if it is down
func (mq *NATS2QueueConnector) CheckConnections() error {
	if !mq.bridge.CheckNATS() {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}
	return nil
}
