package core

import (
	"fmt"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/server/conf"
)

// ConnectToQueueManager utility to connect to a queue manager from a configuration
func ConnectToQueueManager(mqconfig conf.MQConfig) (*ibmmq.MQQueueManager, error) {
	qMgrName := mqconfig.QueueManager

	connectionOptions := ibmmq.NewMQCNO()
	channelDefinition := ibmmq.NewMQCD()

	if mqconfig.UserName != "" {
		connectionSecurityParams := ibmmq.NewMQCSP()
		connectionSecurityParams.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		connectionSecurityParams.UserId = mqconfig.UserName
		connectionSecurityParams.Password = mqconfig.Password

		connectionOptions.SecurityParms = connectionSecurityParams
	}

	if mqconfig.KeyRepository != "" {
		tlsParams := ibmmq.NewMQSCO()
		tlsParams.KeyRepository = mqconfig.KeyRepository
		tlsParams.CertificateLabel = mqconfig.CertificateLabel
		connectionOptions.SSLConfig = tlsParams

		channelDefinition.SSLCipherSpec = "TLS_RSA_WITH_AES_128_CBC_SHA256"
		channelDefinition.SSLPeerName = mqconfig.SSLPeerName
		channelDefinition.CertificateLabel = mqconfig.CertificateLabel
		channelDefinition.SSLClientAuth = int32(ibmmq.MQSCA_REQUIRED)
	}

	channelDefinition.ChannelName = mqconfig.ChannelName
	channelDefinition.ConnectionName = mqconfig.ConnectionName

	connectionOptions.Options = ibmmq.MQCNO_CLIENT_BINDING
	connectionOptions.ClientConn = channelDefinition

	qMgr, err := ibmmq.Connx(qMgrName, connectionOptions)

	if err != nil {
		mqret := err.(*ibmmq.MQReturn)
		if mqret.MQCC == ibmmq.MQCC_WARNING && mqret.MQRC == ibmmq.MQRC_SSL_ALREADY_INITIALIZED {

			// double check the connection went through
			cmho := ibmmq.NewMQCMHO()
			_, err2 := qMgr.CrtMH(cmho)
			if err2 != nil {
				return nil, err
			}

			return &qMgr, nil
		}
		return nil, err
	}

	return &qMgr, nil
}

// SubscribeToChannel uses the bridges STAN connection to subscribe based on the config
// The start position/time and durable name are optional
func (bridge *BridgeServer) SubscribeToChannel(config conf.ConnectorConfig, handler stan.MsgHandler) (stan.Subscription, error) {
	if bridge.Stan() == nil {
		return nil, fmt.Errorf("bridge not configured to use NATS streaming")
	}

	options := []stan.SubscriptionOption{}

	if config.DurableName != "" {
		options = append(options, stan.DurableName(config.DurableName))
	}

	if config.StartAtTime != 0 {
		t := time.Unix(config.StartAtTime, 0)
		options = append(options, stan.StartAtTime(t))
	} else if config.StartAtSequence == -1 {
		options = append(options, stan.StartWithLastReceived())
	} else if config.StartAtSequence > 0 {
		options = append(options, stan.StartAtSequence(uint64(config.StartAtSequence)))
	} else {
		options = append(options, stan.DeliverAllAvailable())
	}

	sub, err := bridge.Stan().Subscribe(config.Channel, handler, options...)

	return sub, err
}
