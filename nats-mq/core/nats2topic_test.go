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
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnNatsReceiveOnTopic(t *testing.T) {
	var topicObject ibmmq.MQObject

	subject := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Topic",
			Subject:        subject,
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	err = tbs.NC.Publish("test", []byte(msg))
	require.NoError(t, err)

	mqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	datalen, err := topicObject.Get(mqmd, gmo, buffer)
	require.NoError(t, err)
	require.Equal(t, msg, string(buffer[:datalen]))

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
	require.Equal(t, int64(len([]byte(msg))), connStats.BytesIn)
	require.Equal(t, int64(len([]byte(msg))), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSendOnNATSReceiveOnTopicMQMD(t *testing.T) {
	var topicObject ibmmq.MQObject

	start := time.Now().UTC()
	subject := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Topic",
			Subject:        subject,
			Topic:          topic,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	bridgeMessage := message.NewBridgeMessage([]byte(msg))
	data, err := bridgeMessage.Encode()
	require.NoError(t, err)

	err = tbs.NC.Publish("test", data)
	require.NoError(t, err)

	mqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	datalen, err := topicObject.Get(mqmd, gmo, buffer)
	require.NoError(t, err)
	require.Equal(t, msg, string(buffer[:datalen]))

	require.Equal(t, start.Format("20060102"), mqmd.PutDate)
	require.True(t, start.Format("15040500") < mqmd.PutTime)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
	require.Equal(t, int64(len(data)), connStats.BytesIn)
	require.Equal(t, int64(datalen), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSimpleSendOnNatsReceiveOnTopicWithTLS(t *testing.T) {
	var topicObject ibmmq.MQObject

	subject := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Topic",
			Subject:        subject,
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTLSTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	err = tbs.NC.Publish("test", []byte(msg))
	require.NoError(t, err)

	mqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	datalen, err := topicObject.Get(mqmd, gmo, buffer)
	require.NoError(t, err)
	require.Equal(t, msg, string(buffer[:datalen]))
}

func TestWildcardSendRecieveOnTopic(t *testing.T) {
	var topicObject ibmmq.MQObject

	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Topic",
			Subject:        "test.*",
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	err = tbs.NC.Publish("test.a", []byte(msg))
	require.NoError(t, err)

	mqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	datalen, err := topicObject.Get(mqmd, gmo, buffer)
	require.NoError(t, err)
	require.Equal(t, msg, string(buffer[:datalen]))
}
