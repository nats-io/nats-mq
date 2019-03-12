package core

import (
	"fmt"
	"github.com/nats-io/go-nats"
	"sync"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

// NATS2TopicConnector connects a NATS subject to an MQ topic
type NATS2TopicConnector struct {
	sync.Mutex

	config ConnectionConfig
	bridge *BridgeServer

	qMgr  *ibmmq.MQQueueManager
	topic *ibmmq.MQObject

	sub *nats.Subscription
}

// NewNATS2TopicConnector create a nats to MQ connector
func NewNATS2TopicConnector(bridge *BridgeServer, config ConnectionConfig) Connector {
	return &NATS2TopicConnector{
		config: config,
		bridge: bridge,
	}
}

func (mq *NATS2TopicConnector) String() string {
	return fmt.Sprintf("NATS:%s to Topic:%s", mq.config.Subject, mq.config.Topic)
}

// Config returns the configuraiton for this connector
func (mq *NATS2TopicConnector) Config() ConnectionConfig {
	return mq.config
}

// Start the connector
func (mq *NATS2TopicConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	if mq.bridge.nats == nil {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
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

	// Create the Object Descriptor that allows us to give the topic name
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT
	mqod.ObjectType = ibmmq.MQOT_TOPIC
	mqod.ObjectString = topicName

	topicObject, err := mq.qMgr.Open(mqod, openOptions)

	if err != nil {
		return err
	}

	mq.topic = &topicObject
	mq.bridge.Logger.Tracef("opened %s", topicName)

	sub, err := mq.bridge.nats.Subscribe(mq.config.Subject, mq.messageHandler)

	if err != nil {
		return err
	}

	mq.sub = sub

	mq.bridge.Logger.Tracef("listening to %s", mq.config.Subject)
	mq.bridge.Logger.Noticef("started connection %s", mq.String())

	return nil
}

func (mq *NATS2TopicConnector) messageHandler(m *nats.Msg) {
	qmgrFlag := mq.qMgr

	if mq.config.ExcludeHeaders {
		qmgrFlag = nil
	}

	mqmd, handle, buffer, err := mq.bridge.natsToMQMessage(m.Data, m.Reply, qmgrFlag)

	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	pmo.OriginalMsgHandle = handle

	// Now put the message to the topic
	err = mq.topic.Put(mqmd, pmo, buffer)

	if err != nil {
		mq.bridge.Logger.Noticef("MQ publish failure, %s, %s", mq.String(), err.Error())
	}
}

// Shutdown the connector
func (mq *NATS2TopicConnector) Shutdown() error {
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
