package core

import (
	"fmt"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// Queue2STANConnector connects an MQ queue to a NATS subject
type Queue2STANConnector struct {
	BridgeConnector

	queue *ibmmq.MQObject
	ctlo  *ibmmq.MQCTLO
}

// NewQueue2STANConnector create a new MQ to Stan connector
func NewQueue2STANConnector(bridge *BridgeServer, config conf.ConnectorConfig) Connector {
	connector := &Queue2STANConnector{}
	connector.init(bridge, config)
	return connector
}

func (mq *Queue2STANConnector) String() string {
	return fmt.Sprintf("Queue:%s to STAN:%s", mq.config.Queue, mq.config.Subject)
}

// Start the connector
func (mq *Queue2STANConnector) Start() error {
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

	// Create the Object Descriptor that allows us to give the queue name
	qObject, err := mq.connectToQueue(mq.config.Queue, ibmmq.MQOO_INPUT_SHARED)
	if err != nil {
		return err
	}

	mq.queue = qObject

	ctlo, err := mq.setUpCallback(mq.queue, mq.stanMessageHandler)
	if err != nil {
		return err
	}
	mq.ctlo = ctlo

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and reading %s", mq.config.Queue)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())
	return nil
}

// Shutdown the connector
func (mq *Queue2STANConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.AddDisconnect()

	mq.bridge.Logger().Noticef("shutting down connection %s", mq.String())

	if mq.ctlo != nil {
		if err := mq.qMgr.Ctl(ibmmq.MQOP_STOP, mq.ctlo); err != nil {
			mq.bridge.Logger().Noticef("unable to stop callbacks for %s", mq.String())
		}
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
