package core

import (
	"fmt"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nats-mq/nats-mq/logging"
)

// Bridge abstracts the bridge server for connectors
type Bridge interface {
	ConnectorError(connector Connector, err error)
	NATS() *nats.Conn
	Stan() stan.Conn
	Logger() logging.Logger
	RegisterReplyInfo(desc string, config conf.ConnectorConfig)

	SubscribeToChannel(config conf.ConnectorConfig, handler stan.MsgHandler) (stan.Subscription, error)
	NATSToMQMessage(data []byte, replyTo string, qmgr *ibmmq.MQQueueManager) (*ibmmq.MQMD, ibmmq.MQMessageHandle, []byte, error)
	MQToNATSMessage(mqmd *ibmmq.MQMD, handle ibmmq.MQMessageHandle, data []byte, length int, qmgr *ibmmq.MQQueueManager) ([]byte, string, error)
}

// Connector is used to type the various connectors in the bridge
type Connector interface {
	Start() error
	Shutdown() error
	String() string
	Config() conf.ConnectorConfig
	Stats() ConnectorStats
}

// CreateConnector builds a connector from the supplied configuration
func CreateConnector(config conf.ConnectorConfig, bridge Bridge) (Connector, error) {
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
