/*
 * Copyright 2012-2019 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
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
			mh, err2 := qMgr.CrtMH(cmho)
			if err2 != nil {
				return nil, err
			}
			mh.DltMH(ibmmq.NewMQDMHO()) // ignore the error

			return &qMgr, nil
		}
		return nil, err
	}

	return &qMgr, nil
}
