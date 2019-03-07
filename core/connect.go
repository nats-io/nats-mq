package core

import (
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"

	nats "github.com/nats-io/go-nats"
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
	natsconfig := bridge.config.NATS

	nc, err := nats.Connect(strings.Join(natsconfig.Servers, ","),
		nats.MaxReconnects(natsconfig.MaxReconnects),
		nats.ReconnectWait(time.Duration(natsconfig.ReconnectWait)*time.Millisecond),
		nats.Timeout(time.Duration(natsconfig.ConnectTimeout)*time.Millisecond),
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
			bridge.Lock()
			if bridge.running {
				bridge.Logger.Debugf("nats connection closed, shutting down bridge...")
				go bridge.Stop()
			}
			bridge.Unlock()
		}))
	if err != nil {
		return err
	}

	bridge.nats = nc

	return nil
}
