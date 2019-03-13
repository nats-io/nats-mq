package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/server/conf"
)

// Queue2NATSConnector connects an MQ queue to a NATS subject
type Queue2NATSConnector struct {
	sync.Mutex

	config conf.ConnectorConfig
	bridge Bridge

	qMgr  *ibmmq.MQQueueManager
	queue *ibmmq.MQObject
	ctlo  *ibmmq.MQCTLO

	stats ConnectorStats
}

// NewQueue2NATSConnector create a new MQ to Stan connector
func NewQueue2NATSConnector(bridge Bridge, config conf.ConnectorConfig) Connector {
	return &Queue2NATSConnector{
		config: config,
		bridge: bridge,
		stats:  NewConnectorStats(),
	}
}

func (mq *Queue2NATSConnector) String() string {
	return fmt.Sprintf("Queue:%s to NATS:%s", mq.config.Queue, mq.config.Subject)
}

// Stats returns a copy of the current stats for this connector
func (mq *Queue2NATSConnector) Stats() ConnectorStats {
	mq.Lock()
	defer mq.Unlock()
	return mq.stats
}

// Config returns the configuraiton for this connector
func (mq *Queue2NATSConnector) Config() conf.ConnectorConfig {
	return mq.config
}

// Start the connector
func (mq *Queue2NATSConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()
	mq.stats.Name = mq.String()

	if mq.bridge.NATS() == nil {
		return fmt.Errorf("%s connector requires nats to be available", mq.String())
	}

	mqconfig := mq.config.MQ
	queueName := mq.config.Queue

	mq.bridge.Logger().Tracef("starting connection %s", mq.String())

	qMgr, err := ConnectToQueueManager(mqconfig)
	if err != nil {
		return err
	}

	mq.bridge.Logger().Tracef("connected to queue manager %s at %s as %s for %s", mqconfig.QueueManager, mqconfig.ConnectionName, mqconfig.ChannelName, mq.String())

	mq.qMgr = qMgr

	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_INPUT_SHARED
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	qObject, err := mq.qMgr.Open(mqod, openOptions)

	if err != nil {
		return err
	}

	mq.queue = &qObject

	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
	//gmo.WaitInterval = mq.config.MQReadTimeout

	cbd := ibmmq.NewMQCBD()
	cbd.CallbackFunction = mq.messageHandler
	err = qObject.CB(ibmmq.MQOP_REGISTER, cbd, getmqmd, gmo)

	if err != nil {
		return err
	}

	mq.ctlo = ibmmq.NewMQCTLO()
	err = mq.qMgr.Ctl(ibmmq.MQOP_START, mq.ctlo)
	if err != nil {
		return err
	}

	mq.stats.AddConnect()
	mq.bridge.Logger().Tracef("opened and reading %s", queueName)
	mq.bridge.Logger().Noticef("started connection %s", mq.String())

	return nil
}

func (mq *Queue2NATSConnector) messageHandler(hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, mqErr *ibmmq.MQReturn) {
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
	natsMsg, replyTo, err := mq.bridge.MQToNATSMessage(md, gmo.MsgHandle, buffer, bufferLen, qmgrFlag)

	if err != nil {
		mq.bridge.Logger().Noticef("failed to convert message for %s, %s", mq.String(), err.Error())
	}

	if replyTo != "" {
		err = mq.bridge.NATS().PublishRequest(mq.config.Subject, replyTo, natsMsg)
	} else {
		err = mq.bridge.NATS().Publish(mq.config.Subject, natsMsg)
	}

	if err != nil {
		mq.bridge.Logger().Noticef("NATS publish failure, %s", mq.String(), err.Error())
		mq.qMgr.Back()
	} else {
		mq.qMgr.Cmit()
		mq.stats.AddMessageOut(int64(len(natsMsg)))
		mq.stats.AddRequestTime(time.Now().Sub(start))
	}
}

// Shutdown the connector
func (mq *Queue2NATSConnector) Shutdown() error {
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
