package core

import (
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"

	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
)

func connectToQueueManager(mqconfig MQConnectionConfig) (*ibmmq.MQQueueManager, error) {
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

func (bridge *BridgeServer) connectToNATS() error {
	bridge.Logger.Noticef("connecting to NATS core...")

	config := bridge.config.NATS

	nc, err := nats.Connect(strings.Join(config.Servers, ","),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(time.Duration(config.ReconnectWait)*time.Millisecond),
		nats.Timeout(time.Duration(config.ConnectTimeout)*time.Millisecond),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			bridge.Logger.Errorf("nats error %s", err.Error())
		}),
		nats.DiscoveredServersHandler(func(nc *nats.Conn) {
			bridge.Logger.Debugf("discovered servers: %v\n", nc.DiscoveredServers())
			bridge.Logger.Debugf("known servers: %v\n", nc.Servers())
		}),
		nats.DisconnectHandler(func(nc *nats.Conn) {
			if !bridge.running { // skip the lock, worst case we print something extra
				return
			}
			bridge.Logger.Debugf("nats connection disconnected...")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			bridge.Logger.Debugf("nats connection reconnected...")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			if bridge.running {
				bridge.Logger.Debugf("nats connection closed, shutting down bridge...")
				bridge.Lock()
				go bridge.Stop()
				bridge.Unlock()
			}
		}))

	if err != nil {
		return err
	}

	bridge.nats = nc
	return nil
}

func (bridge *BridgeServer) connectToSTAN() error {
	bridge.Logger.Noticef("connecting to NATS streaming...")
	config := bridge.config.STAN

	sc, err := stan.Connect(config.ClusterID, config.ClientID,
		stan.NatsConn(bridge.nats),
		stan.PubAckWait(time.Duration(config.PubAckWait)*time.Millisecond),
		stan.MaxPubAcksInflight(config.MaxPubAcksInflight),
		stan.ConnectWait(time.Duration(config.ConnectWait)*time.Millisecond),
		func(o *stan.Options) error {
			o.DiscoverPrefix = config.DiscoverPrefix
			return nil
		})
	if err != nil {
		return err
	}
	bridge.stan = sc

	return nil
}
