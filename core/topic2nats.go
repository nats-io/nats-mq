package core

/*** Started on topic changed to queue
import (
	"context"
	"fmt"
	"sync"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

// Queue2NATSConnector connects an MQ queue to a NATS subject
type Queue2NATSConnector struct {
	sync.Mutex

	config ConnectionConfig
	bridge *BridgeServer

	subscription *ibmmq.MQObject
	topic        *ibmmq.MQObject

	running bool
}

// NewQueue2NATSConnector create a new MQ to Stan connector
func NewQueue2NATSConnector(bridge *BridgeServer, config ConnectionConfig) Connector {
	return &Queue2NATS{
		config: config,
		bridge: bridge,
	}
}

func (mq *Queue2NATSConnector) String() string {
	return fmt.Sprintf("MQ:%s to STAN:%s", mq.config.Topic, mq.config.Channel)
}

// Start the connector
func (mq *Queue2NATSConnector) Start() error {
	mq.Lock()
	defer mq.Unlock()

	if mq.bridge.qMgr == nil {
		return fmt.Errorf("%s connector requires a queue manager to be available", mq.String())
	}

		if mq.bridge.stan == nil {
			return fmt.Errorf("%s connector requires stan to be available", mq.String())
		}

	topicName := mq.config.Topic
	channelName := mq.config.Channel

	mq.bridge.Logger.Tracef("starting connection from to MQ:%s to STAN:%s", topicName, channelName)

	mq.topic = &ibmmq.MQObject{} // have to create this before calling sub

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topicName
	subscriptionObject, err := mq.bridge.qMgr.Sub(mqsd, mq.topic)

	if err != nil {
		return err
	}

	mq.subscription = &subscriptionObject

	mq.bridge.Logger.Tracef("subscribed to %s", topicName)
	mq.bridge.Logger.Noticef("started connection from to MQ:%s to STAN:%s", topicName, channelName)

	return nil
}

func (mq *Queue2NATSConnector) listen() {
	var err error

	running := true
	for running == true && err == nil {
		var datalen int

		// The GET requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// The default options are OK, but it's always
		// a good idea to be explicit about transactional boundaries as
		// not all platforms behave the same way.
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT

		// Set options to wait for a maximum of 3 seconds for any new message to arrive
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample.
		buffer := make([]byte, 1024)

		// Now we can try to get the message
		datalen, err = qObject.Get(getmqmd, gmo, buffer)

		if err != nil {
			running = false
			fmt.Println(err)
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				// If there's no message available, then I won't treat that as a real error as
				// it's an expected situation
				err = nil
			}
		} else {
			// Assume the message is a printable string, which it will be
			// if it's been created by the amqspub program
			fmt.Printf("Got message of length %d: ", datalen)
			fmt.Println(strings.TrimSpace(string(buffer[:datalen])))
		}
	}
}

// Shutdown the connector
func (mq *Queue2NATSConnector) Shutdown(context context.Context) error {
	mq.Lock()
	defer mq.Unlock()

	if mq.subscription == nil {
		return nil
	}

	mq.bridge.Logger.Noticef("shutting down connection from MQ:%s to STAN:%s", mq.config.Topic, mq.config.Channel)
	err := mq.subscription.Close(0)

	mq.subscription = nil
	mq.topic = nil
	return err
}
*/
