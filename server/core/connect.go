package core

import (
	"log"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/server/conf"
)

// ConnectToQueueManager utility to connect to a queue manager from a configuration
func ConnectToQueueManager(mqconfig conf.MQConfig) (*ibmmq.MQQueueManager, error) {
	connectionOptions := ibmmq.NewMQCNO()
	channelDefinition := ibmmq.NewMQCD()

	qMgrName := mqconfig.QueueManager
	channelDefinition.ChannelName = mqconfig.ChannelName
	channelDefinition.ConnectionName = mqconfig.ConnectionName

	connectionOptions.ClientConn = channelDefinition
	connectionOptions.Options = ibmmq.MQCNO_CLIENT_BINDING

	if mqconfig.UserName != "" {
		connectionSecurityParams := ibmmq.NewMQCSP()
		connectionSecurityParams.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		connectionSecurityParams.UserId = mqconfig.UserName
		connectionSecurityParams.Password = mqconfig.Password
		connectionOptions.SecurityParms = connectionSecurityParams
	}

	qMgr, err := ibmmq.Connx(qMgrName, connectionOptions)

	if err != nil {
		return nil, err
	}

	return &qMgr, nil
}

// ConnectToSTANWithConfig utility to connect to a streaming server from a config
// unused by the bridge
func ConnectToSTANWithConfig(config conf.NATSStreamingConfig, nc *nats.Conn) (stan.Conn, error) {
	sc, err := stan.Connect(config.ClusterID, config.ClientID,
		stan.NatsConn(nc),
		stan.PubAckWait(time.Duration(config.PubAckWait)*time.Millisecond),
		stan.MaxPubAcksInflight(config.MaxPubAcksInflight),
		stan.ConnectWait(time.Duration(config.ConnectWait)*time.Millisecond),
		func(o *stan.Options) error {
			o.DiscoverPrefix = config.DiscoverPrefix
			return nil
		})
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// ConnectToNATSWithConfig utility to connect to nats from a config
// unused by the bridge, which uses its own logger, this method uses "log"
func ConnectToNATSWithConfig(config conf.NATSConfig) (*nats.Conn, error) {
	nc, err := nats.Connect(strings.Join(config.Servers, ","),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(time.Duration(config.ReconnectWait)*time.Millisecond),
		nats.Timeout(time.Duration(config.ConnectTimeout)*time.Millisecond),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("nats error %s", err.Error())
		}),
		nats.DiscoveredServersHandler(func(nc *nats.Conn) {
			log.Printf("discovered servers: %v\n", nc.DiscoveredServers())
			log.Printf("known servers: %v\n", nc.Servers())
		}),
		nats.DisconnectHandler(func(nc *nats.Conn) {
			log.Printf("nats connection disconnected...")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("nats connection reconnected...")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("nats connection closed...")
		}),
	)
	return nc, err
}
