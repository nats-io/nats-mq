package core

import (
	"fmt"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// Topic2StanConnector connects an MQ queue to a NATS channel
type Topic2StanConnector struct {
	BridgeConnector

	ctlo  *ibmmq.MQCTLO
	topic *ibmmq.MQObject
	sub   *ibmmq.MQObject
}

// NewTopic2StanConnector create a new MQ to Stan connector
func NewTopic2StanConnector(bridge *BridgeServer, config conf.ConnectorConfig) Connector {
	connector := &Topic2StanConnector{}
	connector.init(bridge, config, fmt.Sprintf("MQ:%s to Stan:%s", config.Topic, config.Channel))
	return connector
}

// Start the connector
func (mq *Topic2StanConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	if mq.bridge.Stan() == nil {
		return fmt.Errorf("%s connector requires nats streaming to be available", mq.String())
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

	ctlo, err := mq.setUpCallback(mq.topic, mq.stanMessageHandler)
	if err != nil {
		return err
	}
	mq.ctlo = ctlo

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and subscribed to %s", mq.config.Topic)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

// Shutdown the connector
func (mq *Topic2StanConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()

	mq.stats.AddDisconnect()

	if mq.topic == nil {
		return nil
	}

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	if mq.ctlo != nil {
		if err := mq.qMgr.Ctl(ibmmq.MQOP_STOP, mq.ctlo); err != nil {
			mq.bridge.Logger().Noticef("unable to stop callbacks for %s", mq.String())
		}
	}

	var err error
	var err2 error

	sub := mq.sub
	topic := mq.topic
	mq.topic = nil
	mq.sub = nil

	if sub != nil {
		err = sub.Close(0)

		if err != nil {
			mq.bridge.Logger().Noticef("error closing subscription for %s", mq.String())
		}
	}

	if topic != nil {
		err2 = topic.Close(0)

		if err2 != nil {
			mq.bridge.Logger().Noticef("error closing topic for %s", mq.String())
		}
	}

	// Return an error if either errored, but at this point not much we can do
	if err == nil {
		err = err2
	}

	if mq.qMgr != nil {
		_ = mq.qMgr.Disc()
		mq.qMgr = nil
		mq.bridge.Logger().Tracef("disconnected from queue manager for %s", mq.String())
	}

	return nil //err // ignore the disconnect error
}
