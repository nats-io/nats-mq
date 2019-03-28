package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nuid"
)

// Connector is the abstraction for all of the bridge connector types
type Connector interface {
	Start() error
	Shutdown() error

	CheckConnections() error

	String() string
	ID() string

	Stats() ConnectorStats
}

// CreateConnector builds a connector from the supplied configuration
func CreateConnector(config conf.ConnectorConfig, bridge *BridgeServer) (Connector, error) {
	switch config.Type {
	case conf.Queue2NATS:
		bridge.RegisterReplyInfo("S:"+config.Subject, config)
		return NewQueue2NATSConnector(bridge, config), nil
	case conf.Queue2Stan:
		bridge.RegisterReplyInfo("C:"+config.Channel, config)
		return NewQueue2STANConnector(bridge, config), nil
	case conf.NATS2Queue:
		bridge.RegisterReplyInfo("Q:"+config.Queue+"@"+config.MQ.QueueManager, config)
		return NewNATS2QueueConnector(bridge, config), nil
	case conf.Stan2Queue:
		bridge.RegisterReplyInfo("Q:"+config.Queue+"@"+config.MQ.QueueManager, config)
		return NewStan2QueueConnector(bridge, config), nil
	case conf.Topic2NATS:
		bridge.RegisterReplyInfo("S:"+config.Subject, config)
		return NewTopic2NATSConnector(bridge, config), nil
	case conf.Topic2Stan:
		bridge.RegisterReplyInfo("C:"+config.Channel, config)
		return NewTopic2StanConnector(bridge, config), nil
	case conf.NATS2Topic:
		bridge.RegisterReplyInfo("T:"+config.Topic+"@"+config.MQ.QueueManager, config)
		return NewNATS2TopicConnector(bridge, config), nil
	case conf.Stan2Topic:
		bridge.RegisterReplyInfo("T:"+config.Topic+"@"+config.MQ.QueueManager, config)
		return NewStan2TopicConnector(bridge, config), nil
	default:
		return nil, fmt.Errorf("unknown connector type %q in configuration", config.Type)
	}
}

// BridgeConnector is the base type used for connectors so that they can share code
type BridgeConnector struct {
	sync.Mutex

	config conf.ConnectorConfig
	bridge *BridgeServer
	stats  ConnectorStats

	qMgr *ibmmq.MQQueueManager

	inflight       int
	inflightKicker <-chan time.Time
}

// Start is a no-op, designed for overriding
func (mq *BridgeConnector) Start() error {
	return nil
}

// Shutdown is a no-op, designed for overriding
func (mq *BridgeConnector) Shutdown() error {
	return nil
}

// CheckConnections is a no-op, designed for overriding
// This is called when nats or stan goes down
// the connector should return an error if it has to be shut down
func (mq *BridgeConnector) CheckConnections() error {
	return nil
}

// String returns the name passed into init
func (mq *BridgeConnector) String() string {
	return mq.stats.Name
}

// ID returns the id from the stats
func (mq *BridgeConnector) ID() string {
	return mq.stats.ID
}

// Stats returns a copy of the current stats for this connector
func (mq *BridgeConnector) Stats() ConnectorStats {
	mq.Lock()
	defer mq.Unlock()
	return mq.stats
}

// Init sets up common fields for all connectors
func (mq *BridgeConnector) init(bridge *BridgeServer, config conf.ConnectorConfig, name string) {
	mq.config = config
	mq.bridge = bridge
	mq.stats = NewConnectorStats()
	mq.stats.Name = name
	mq.stats.ID = mq.config.ID

	if mq.config.ID == "" {
		mq.stats.ID = nuid.Next()
	}
}

// init the MQ connection - expects the lock to be held by the caller
func (mq *BridgeConnector) connectToMQ() error {
	mqconfig := mq.config.MQ

	qMgr, err := ConnectToQueueManager(mqconfig)
	if err != nil {
		return err
	}

	mq.bridge.Logger().Tracef("connected to queue manager %s at %s as %s for %s", mqconfig.QueueManager, mqconfig.ConnectionName, mqconfig.ChannelName, mq.String())

	mq.qMgr = qMgr
	return nil
}

func (mq *BridgeConnector) connectToQueue(queueName string, openOptions int32) (*ibmmq.MQObject, error) {
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	qObject, err := mq.qMgr.Open(mqod, openOptions)

	if err != nil {
		return nil, err
	}

	return &qObject, nil
}

// subscribeToTopic subscribes to a topic
func (mq *BridgeConnector) subscribeToTopic(topicName string) (*ibmmq.MQObject, *ibmmq.MQObject, error) {
	topic := &ibmmq.MQObject{}
	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topicName
	subscriptionObject, err := mq.qMgr.Sub(mqsd, topic)

	if err != nil {
		return nil, nil, err
	}

	return topic, &subscriptionObject, nil
}

// connectToTopic sets up a topic for output
func (mq *BridgeConnector) connectToTopic(topicName string) (*ibmmq.MQObject, error) {
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT
	mqod.ObjectType = ibmmq.MQOT_TOPIC
	mqod.ObjectString = topicName
	topic, err := mq.qMgr.Open(mqod, openOptions)
	if err != nil {
		return nil, err
	}
	return &topic, err
}

// NATSCallback used by mq-nats connectors in an MQ library callback
// The lock will be held by the caller!
type NATSCallback func(natsMsg []byte, replyTo string) error

// ShutdownCallback is returned when setting up a callback or polling so the connector can shut it down
type ShutdownCallback func() error

func (mq *BridgeConnector) setUpListener(target *ibmmq.MQObject, cb NATSCallback, conn Connector) (ShutdownCallback, error) {
	if mq.config.UsePolling {
		return mq.setUpPolling(target, cb, conn)
	}
	return mq.setUpCallback(target, cb, conn)
}

func (mq *BridgeConnector) setUpCallback(target *ibmmq.MQObject, cb NATSCallback, conn Connector) (ShutdownCallback, error) {
	mqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	cmho := ibmmq.NewMQCMHO()
	propsMsgHandle, err := mq.qMgr.CrtMH(cmho)

	if err != nil {
		return nil, err
	}

	gmo.MsgHandle = propsMsgHandle
	gmo.Options = ibmmq.MQGMO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
	gmo.Options |= ibmmq.MQGMO_PROPERTIES_IN_HANDLE

	mq.bridge.Logger().Tracef("setting up callback for %s", mq.String())

	cbd := ibmmq.NewMQCBD()
	cbd.CallbackFunction = mq.createMQCallback(cb, conn)

	err = target.CB(ibmmq.MQOP_REGISTER, cbd, mqmd, gmo)

	if err != nil {
		return nil, err
	}

	ctlo := ibmmq.NewMQCTLO()
	ctlo.Options = ibmmq.MQCTLO_FAIL_IF_QUIESCING
	err = mq.qMgr.Ctl(ibmmq.MQOP_START, ctlo)
	if err != nil {
		return nil, err
	}

	return func() error {
		if err := mq.qMgr.Ctl(ibmmq.MQOP_STOP, ctlo); err != nil {
			mq.bridge.Logger().Noticef("error stopping callbacks, %s", err.Error())
		}
		gmo.MsgHandle.DltMH(ibmmq.NewMQDMHO()) // ignore the error
		return nil
	}, nil
}

func (mq *BridgeConnector) setUpPolling(target *ibmmq.MQObject, cb NATSCallback, conn Connector) (ShutdownCallback, error) {
	bufferSize := mq.config.IncomingBufferSize
	if bufferSize == 0 {
		bufferSize = 1024 * 8
	}
	buffer := make([]byte, bufferSize)

	waitTimeout := int32(mq.config.IncomingMessageWait)
	if waitTimeout == 0 {
		waitTimeout = int32(500)
	}
	running := true
	done := make(chan bool)
	callback := mq.createMQCallback(cb, conn)

	cmho := ibmmq.NewMQCMHO()
	propsMsgHandle, err := mq.qMgr.CrtMH(cmho)

	if err != nil {
		return nil, err
	}

	mq.bridge.Logger().Tracef("starting polling for %s", mq.String())

	go func() {
		for running {
			mqmd := ibmmq.NewMQMD()
			gmo := ibmmq.NewMQGMO()
			gmo.Options = ibmmq.MQGMO_SYNCPOINT
			gmo.Options |= ibmmq.MQGMO_WAIT
			gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
			gmo.Options |= ibmmq.MQGMO_PROPERTIES_IN_HANDLE
			gmo.MsgHandle = propsMsgHandle
			gmo.WaitInterval = waitTimeout

			len, err := target.Get(mqmd, gmo, buffer)

			if err != nil {
				mqret := err.(*ibmmq.MQReturn)
				if mqret.MQRC != ibmmq.MQRC_NO_MSG_AVAILABLE {
					callback(mq.qMgr, target, mqmd, gmo, buffer[0:len], nil, mqret)
				}
			} else {
				callback(mq.qMgr, target, mqmd, gmo, buffer[0:len], nil, nil)
			}

			select {
			case <-done:
				running = false
			default:
			}
		}

		propsMsgHandle.DltMH(ibmmq.NewMQDMHO()) // ignore the error
	}()

	return func() error {
		close(done)
		return nil
	}, nil
}

func (mq *BridgeConnector) createMQCallback(cb NATSCallback, conn Connector) func(qMgr *ibmmq.MQQueueManager, hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, mqErr *ibmmq.MQReturn) {
	return func(qMgr *ibmmq.MQQueueManager, hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, mqErr *ibmmq.MQReturn) {
		mq.Lock()
		defer mq.Unlock()
		start := time.Now()

		if mqErr != nil && mqErr.MQCC != ibmmq.MQCC_OK {
			if mqErr.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				mq.bridge.Logger().Tracef("message timeout on %s", mq.String())
				return
			}

			err := fmt.Errorf("mq error in callback %s", mqErr.Error())
			go mq.bridge.ConnectorError(conn, err)
			return
		}

		// ignore event calls
		if cbc != nil && cbc.CallType == ibmmq.MQCBCT_EVENT_CALL {
			return
		}

		bufferLen := len(buffer)

		mq.bridge.Logger().Tracef("%s got raw mq message with body of length %d", mq.String(), bufferLen)

		qmgrFlag := mq.qMgr

		if mq.config.ExcludeHeaders {
			qmgrFlag = nil
		}

		mq.stats.AddMessageIn(int64(bufferLen))
		natsMsg, replyTo, err := mq.bridge.MQToNATSMessage(md, gmo.MsgHandle, buffer, bufferLen, qmgrFlag)

		if err != nil {
			mq.bridge.Logger().Noticef("message conversion failure %s, %s", mq.String(), err.Error())
			mq.qMgr.Back()
			return
		}

		err = cb(natsMsg, replyTo)

		if err != nil {
			mq.bridge.Logger().Noticef("publish failure for %s, %s", mq.String(), err.Error())
			mq.qMgr.Back()
		} else {
			mq.inflight++
			maxInFlight := mq.config.MaxMQMessagesInFlight
			if maxInFlight <= 0 || mq.inflight == maxInFlight {
				if err := mq.qMgr.Cmit(); err != nil {
					mq.bridge.Logger().Noticef("failed to commit, %s", err.Error())
					go mq.bridge.ConnectorError(conn, err) // run in a go routine so we can finish this method and unlock
					return
				}
				mq.inflight = 0
			} else if mq.inflightKicker == nil {
				mq.inflightKicker = time.After(250 * time.Millisecond)
				go func(mq *BridgeConnector) {
					mq.Lock()
					kicker := mq.inflightKicker
					mq.Unlock()

					if kicker == nil {
						return
					}

					<-kicker

					mq.Lock()
					if mq.inflight > 0 && mq.qMgr != nil {
						ctlo := ibmmq.NewMQCTLO()
						ctlo.Options = ibmmq.MQCTLO_FAIL_IF_QUIESCING
						if err := mq.qMgr.Ctl(ibmmq.MQOP_SUSPEND, ctlo); err != nil {
							mq.bridge.Logger().Noticef("failed to suspend for commit in kicker, %s", err.Error())
							go mq.bridge.ConnectorError(conn, err) // run in a go routine so we can finish this method and unlock
						} else if err := mq.qMgr.Cmit(); err != nil {
							mq.bridge.Logger().Noticef("failed to commit in kicker, %s", err.Error())
							go mq.bridge.ConnectorError(conn, err) // run in a go routine so we can finish this method and unlock
						} else if err := mq.qMgr.Ctl(ibmmq.MQOP_RESUME, ctlo); err != nil {
							mq.bridge.Logger().Noticef("failed to resume after commit in kicker, %s", err.Error())
							go mq.bridge.ConnectorError(conn, err) // run in a go routine so we can finish this method and unlock
						}
					}
					mq.inflight = 0
					mq.inflightKicker = nil
					mq.Unlock()
				}(mq)
			}
			mq.stats.AddMessageOut(int64(len(natsMsg)))
			mq.stats.AddRequestTime(time.Since(start))
		}
	}
}

func (mq *BridgeConnector) stanMessageHandler(natsMsg []byte, replyTo string) error {
	return mq.bridge.Stan().Publish(mq.config.Channel, natsMsg)
}

func (mq *BridgeConnector) natsMessageHandler(natsMsg []byte, replyTo string) error {
	var err error
	if replyTo != "" {
		err = mq.bridge.NATS().PublishRequest(mq.config.Subject, replyTo, natsMsg)
	} else {
		err = mq.bridge.NATS().Publish(mq.config.Subject, natsMsg)
	}
	return err
}

// set up a nats subscription, assumes the lock is held
func (mq *BridgeConnector) subscribeToNATS(subject string, dest *ibmmq.MQObject) (*nats.Subscription, error) {
	return mq.bridge.NATS().Subscribe(subject, func(m *nats.Msg) {
		mq.Lock()
		defer mq.Unlock()
		start := time.Now()

		qmgrFlag := mq.qMgr

		if mq.config.ExcludeHeaders {
			qmgrFlag = nil
		}
		mq.stats.AddMessageIn(int64(len(m.Data)))
		mqmd, handle, buffer, err := mq.bridge.NATSToMQMessage(m.Data, m.Reply, qmgrFlag)

		mq.bridge.Logger().Tracef("%s got decoded nats message with body length %d", mq.String(), len(buffer))

		if err != nil {
			mq.bridge.Logger().Noticef("message conversion failure, %s, %s", mq.String(), err.Error())
			return
		}

		pmo := ibmmq.NewMQPMO()
		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
		pmo.OriginalMsgHandle = handle

		err = dest.Put(mqmd, pmo, buffer)

		if err != nil {
			mq.bridge.Logger().Noticef("MQ publish failure, %s, %s", mq.String(), err.Error())
		} else {
			mq.stats.AddMessageOut(int64(len(buffer)))
			mq.stats.AddRequestTime(time.Since(start))
		}
	})
}

// subscribeToChannel uses the bridges STAN connection to subscribe based on the config
// The start position/time and durable name are optional
func (mq *BridgeConnector) subscribeToChannel(dest *ibmmq.MQObject) (stan.Subscription, error) {
	if mq.bridge.Stan() == nil {
		return nil, fmt.Errorf("bridge not configured to use NATS streaming")
	}

	options := []stan.SubscriptionOption{}

	if mq.config.DurableName != "" {
		options = append(options, stan.DurableName(mq.config.DurableName))
	}

	if mq.config.StartAtTime != 0 {
		t := time.Unix(mq.config.StartAtTime, 0)
		options = append(options, stan.StartAtTime(t))
	} else if mq.config.StartAtSequence == -1 {
		options = append(options, stan.StartWithLastReceived())
	} else if mq.config.StartAtSequence > 0 {
		options = append(options, stan.StartAtSequence(uint64(mq.config.StartAtSequence)))
	} else {
		options = append(options, stan.DeliverAllAvailable())
	}

	options = append(options, stan.SetManualAckMode())

	sub, err := mq.bridge.Stan().Subscribe(mq.config.Channel, func(msg *stan.Msg) {
		mq.Lock()
		defer mq.Unlock()
		start := time.Now()

		qmgrFlag := mq.qMgr

		if mq.config.ExcludeHeaders {
			qmgrFlag = nil
		}

		mq.stats.AddMessageIn(int64(len(msg.Data)))
		mqmd, handle, buffer, err := mq.bridge.NATSToMQMessage(msg.Data, "", qmgrFlag)
		if err != nil {
			mq.bridge.Logger().Noticef("message conversion failure, %s, %s", mq.String(), err.Error())
			return
		}
		mq.bridge.Logger().Tracef("%s got decoded stan message with body length %d", mq.String(), len(buffer))

		pmo := ibmmq.NewMQPMO()
		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
		pmo.OriginalMsgHandle = handle

		err = dest.Put(mqmd, pmo, buffer)

		if err != nil {
			mq.bridge.Logger().Noticef("MQ put failure, %s, %s", mq.String(), err.Error())
		} else {
			msg.Ack()
			mq.stats.AddMessageOut(int64(len(buffer)))
			mq.stats.AddRequestTime(time.Since(start))
		}
	}, options...)

	return sub, err
}
