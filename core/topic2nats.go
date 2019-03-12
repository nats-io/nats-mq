package core

import (
	"fmt"
	"sync"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

// Topic2NATSConnector connects an MQ queue to a NATS subject
type Topic2NATSConnector struct {
	sync.Mutex

	config ConnectionConfig
	bridge *BridgeServer

	qMgr  *ibmmq.MQQueueManager
	topic *ibmmq.MQObject
	sub   *ibmmq.MQObject
	ctlo  *ibmmq.MQCTLO
}

// NewTopic2NATSConnector create a new MQ to Stan connector
func NewTopic2NATSConnector(bridge *BridgeServer, config ConnectionConfig) Connector {
	return &Topic2NATSConnector{
		config: config,
		bridge: bridge,
	}
}

func (mq *Topic2NATSConnector) String() string {
	return fmt.Sprintf("MQ:%s to NATS:%s", mq.config.Topic, mq.config.Subject)
}

// Config returns the configuraiton for this connector
func (mq *Topic2NATSConnector) Config() ConnectionConfig {
	return mq.config
}

// Start the connector
func (mq *Topic2NATSConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	if mq.bridge.nats == nil {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}

	mqconfig := mq.config.MQ
	topicName := mq.config.Topic

	mq.bridge.Logger.Tracef("starting connection %s", mq.String())

	qMgr, err := connectToQueueManager(mqconfig)
	if err != nil {
		return err
	}

	mq.bridge.Logger.Tracef("connected to queue manager %s at %s as %s for %s", mqconfig.QueueManager, mqconfig.ConnectionName, mqconfig.ChannelName, mq.String())

	mq.qMgr = qMgr
	mq.topic = &ibmmq.MQObject{}

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topicName
	subscriptionObject, err := mq.qMgr.Sub(mqsd, mq.topic)

	if err != nil {
		return err
	}

	mq.sub = &subscriptionObject

	mq.bridge.Logger.Tracef("subscribed to %s", topicName)

	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING

	cbd := ibmmq.NewMQCBD()
	cbd.CallbackFunction = mq.messageHandler

	mq.bridge.Logger.Tracef("registering callback for %s", mq.String())
	err = mq.topic.CB(ibmmq.MQOP_REGISTER, cbd, getmqmd, gmo)

	if err != nil {
		return err
	}

	mq.bridge.Logger.Tracef("starting callback for %s", mq.String())
	mq.ctlo = ibmmq.NewMQCTLO()
	err = mq.qMgr.Ctl(ibmmq.MQOP_START, mq.ctlo)
	if err != nil {
		return err
	}

	mq.bridge.Logger.Tracef("opened and subscribed to %s", topicName)
	mq.bridge.Logger.Noticef("started connection %s", mq.String())

	return nil
}

func (mq *Topic2NATSConnector) messageHandler(hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, mqErr *ibmmq.MQReturn) {
	if mqErr != nil && mqErr.MQCC != ibmmq.MQCC_OK {
		if mqErr.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
			mq.bridge.Logger.Tracef("message timeout on %s", mq.String())
			return
		}

		err := fmt.Errorf("mq error in callback %s", mqErr.Error())
		go mq.bridge.connectorError(mq, err)
		return
	}

	bufferLen := len(buffer)

	mq.bridge.Logger.Tracef("%s got message of length %d", mq.String(), bufferLen)

	qmgrFlag := mq.qMgr

	if mq.config.ExcludeHeaders {
		qmgrFlag = nil
	}

	natsMsg, replyTo, err := mq.bridge.mqToNATSMessage(md, gmo.MsgHandle, buffer, bufferLen, qmgrFlag)

	if err != nil {
		mq.bridge.Logger.Noticef("failed to convert message for %s, %s", mq.String(), err.Error())
	}

	if replyTo != "" {
		err = mq.bridge.nats.PublishRequest(mq.config.Subject, replyTo, natsMsg)
	} else {
		err = mq.bridge.nats.Publish(mq.config.Subject, natsMsg)
	}

	if err != nil {
		mq.bridge.Logger.Noticef("NATS publish failure, %s", mq.String(), err.Error())
		mq.qMgr.Back()
	} else {
		mq.qMgr.Cmit()
	}
}

// Shutdown the connector
func (mq *Topic2NATSConnector) Shutdown() error {
	mq.Lock()
	defer mq.Unlock()

	if mq.topic == nil {
		return nil
	}

	mq.bridge.Logger.Noticef("shutting down connection %s", mq.String())

	if mq.ctlo != nil {
		if err := mq.qMgr.Ctl(ibmmq.MQOP_STOP, mq.ctlo); err != nil {
			mq.bridge.Logger.Noticef("unable to stop callbacks for %s", mq.String())
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
			mq.bridge.Logger.Noticef("error closing subscription for %s", mq.String())
		}
	}

	if topic != nil {
		err2 = topic.Close(0)

		if err2 != nil {
			mq.bridge.Logger.Noticef("error closing topic for %s", mq.String())
		}
	}

	// Return an error if either errored, but at this point not much we can do
	if err == nil {
		err = err2
	}

	if mq.qMgr != nil {
		_ = mq.qMgr.Disc()
		mq.bridge.Logger.Tracef("disconnected from queue manager for %s", mq.String())
	}

	return nil //err // ignore the disconnect error
}
