package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
)

// Topic2StanConnector connects an MQ queue to a NATS channel
type Topic2StanConnector struct {
	sync.Mutex

	config conf.ConnectorConfig
	bridge Bridge

	qMgr  *ibmmq.MQQueueManager
	topic *ibmmq.MQObject
	sub   *ibmmq.MQObject
	ctlo  *ibmmq.MQCTLO

	stats ConnectorStats
}

// NewTopic2StanConnector create a new MQ to Stan connector
func NewTopic2StanConnector(bridge Bridge, config conf.ConnectorConfig) Connector {
	return &Topic2StanConnector{
		config: config,
		bridge: bridge,
		stats:  NewConnectorStats(),
	}
}

func (mq *Topic2StanConnector) String() string {
	return fmt.Sprintf("MQ:%s to Stan:%s", mq.config.Topic, mq.config.Channel)
}

// Stats returns a copy of the current stats for this connector
func (mq *Topic2StanConnector) Stats() ConnectorStats {
	mq.Lock()
	defer mq.Unlock()
	return mq.stats
}

// Config returns the configuraiton for this connector
func (mq *Topic2StanConnector) Config() conf.ConnectorConfig {
	return mq.config
}

// Start the connector
func (mq *Topic2StanConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	mq.stats.Name = mq.String()

	if mq.bridge.NATS() == nil {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}

	mqconfig := mq.config.MQ
	topicName := mq.config.Topic

	mq.bridge.Logger().Tracef("starting connection %s", mq.String())

	qMgr, err := ConnectToQueueManager(mqconfig)
	if err != nil {
		return err
	}

	mq.bridge.Logger().Tracef("connected to queue manager %s at %s as %s for %s", mqconfig.QueueManager, mqconfig.ConnectionName, mqconfig.ChannelName, mq.String())

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

	mq.bridge.Logger().Tracef("subscribed to %s", topicName)

	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING

	cbd := ibmmq.NewMQCBD()
	cbd.CallbackFunction = mq.messageHandler

	mq.bridge.Logger().Tracef("registering callback for %s", mq.String())
	err = mq.topic.CB(ibmmq.MQOP_REGISTER, cbd, getmqmd, gmo)

	if err != nil {
		return err
	}

	mq.bridge.Logger().Tracef("starting callback for %s", mq.String())
	mq.ctlo = ibmmq.NewMQCTLO()
	err = mq.qMgr.Ctl(ibmmq.MQOP_START, mq.ctlo)
	if err != nil {
		return err
	}

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and subscribed to %s", topicName)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

func (mq *Topic2StanConnector) messageHandler(hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, mqErr *ibmmq.MQReturn) {
	mq.Lock()
	defer mq.Unlock()
	start := time.Now()

	if mqErr != nil && mqErr.MQCC != ibmmq.MQCC_OK {
		if mqErr.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
			mq.bridge.Logger().Tracef("message timeout on %s", mq.String())
			return
		}

		err := fmt.Errorf("mq error in callback %s", mqErr.Error())
		go mq.bridge.ConnectorError(mq, err)
		return
	}

	bufferLen := len(buffer)

	mq.bridge.Logger().Tracef("%s got message of length %d", mq.String(), bufferLen)

	qmgrFlag := mq.qMgr

	if mq.config.ExcludeHeaders {
		qmgrFlag = nil
	}

	mq.stats.AddMessageIn(int64(bufferLen))
	natsMsg, _, err := mq.bridge.MQToNATSMessage(md, gmo.MsgHandle, buffer, bufferLen, qmgrFlag)

	if err != nil {
		mq.bridge.Logger().Noticef("failed to convert message for %s, %s", mq.String(), err.Error())
	}

	err = mq.bridge.Stan().Publish(mq.config.Channel, natsMsg)

	if err != nil {
		mq.bridge.Logger().Noticef("STAN publish failure, %s", mq.String(), err.Error())
		mq.qMgr.Back()
	} else {
		mq.qMgr.Cmit()
		mq.stats.AddMessageOut(int64(len(natsMsg)))
		mq.stats.AddRequestTime(time.Now().Sub(start))
	}
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
