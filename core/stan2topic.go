package core

import (
	"fmt"
	"github.com/nats-io/go-nats-streaming"
	"sync"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

// Stan2TopicConnector connects a STAN channel to an MQ Topic
type Stan2TopicConnector struct {
	sync.Mutex

	config ConnectorConfig
	bridge *BridgeServer

	qMgr  *ibmmq.MQQueueManager
	topic *ibmmq.MQObject

	sub stan.Subscription
}

// NewStan2TopicConnector create a new Stan to MQ connector
func NewStan2TopicConnector(bridge *BridgeServer, config ConnectorConfig) Connector {
	return &Stan2TopicConnector{
		config: config,
		bridge: bridge,
	}
}

func (mq *Stan2TopicConnector) String() string {
	return fmt.Sprintf("STAN:%s to Topic:%s", mq.config.Channel, mq.config.Topic)
}

// Config returns the configuraiton for this connector
func (mq *Stan2TopicConnector) Config() ConnectorConfig {
	return mq.config
}

// Start the connector
func (mq *Stan2TopicConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	if mq.bridge.stan == nil {
		return fmt.Errorf("%s connector requires nats streaming to be available", mq.String())
	}

	mqconfig := mq.config.MQ
	topicName := mq.config.Topic

	mq.bridge.Logger.Tracef("starting connection %s", mq.String())

	qMgr, err := ConnectToQueueManager(mqconfig)
	if err != nil {
		return err
	}

	mq.bridge.Logger.Tracef("connected to queue manager %s at %s as %s for %s", mqconfig.QueueManager, mqconfig.ConnectionName, mqconfig.ChannelName, mq.String())

	mq.qMgr = qMgr

	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT
	mqod.ObjectType = ibmmq.MQOT_TOPIC
	mqod.ObjectString = topicName

	qObject, err := mq.qMgr.Open(mqod, openOptions)

	if err != nil {
		return err
	}

	mq.topic = &qObject
	mq.bridge.Logger.Tracef("opened %s", topicName)

	sub, err := mq.bridge.stan.Subscribe(mq.config.Channel, mq.messageHandler)

	if err != nil {
		return err
	}

	mq.sub = sub

	mq.bridge.Logger.Tracef("reading %s", mq.config.Channel)
	mq.bridge.Logger.Noticef("started connection %s", mq.String())

	return nil
}

func (mq *Stan2TopicConnector) messageHandler(m *stan.Msg) {
	mq.Lock()
	defer mq.Unlock()

	qmgrFlag := mq.qMgr

	if mq.config.ExcludeHeaders {
		qmgrFlag = nil
	}

	mqmd, handle, buffer, err := mq.bridge.natsToMQMessage(m.Data, "", qmgrFlag)

	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	pmo.OriginalMsgHandle = handle

	// Now put the message to the queue
	err = mq.topic.Put(mqmd, pmo, buffer)

	if err != nil {
		mq.bridge.Logger.Noticef("MQ publish failure, %s, %s", mq.String(), err.Error())
	}
}

// Shutdown the connector
func (mq *Stan2TopicConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()

	mq.bridge.Logger.Noticef("shutting down connection %s", mq.String())

	var err error

	topic := mq.topic
	mq.topic = nil

	if topic != nil {
		err = topic.Close(0)
	}

	if mq.qMgr != nil {
		_ = mq.qMgr.Disc()
		mq.bridge.Logger.Tracef("disconnected from queue manager for %s", mq.String())
	}

	if mq.sub != nil {
		mq.sub.Unsubscribe()
		mq.sub = nil
	}

	return err // ignore the disconnect error
}
