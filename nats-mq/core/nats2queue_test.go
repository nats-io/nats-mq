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
	"bytes"
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnNatsReceiveOnQueue(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Queue",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	err = tbs.NC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

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

func TestSendOnNATSReceiveOnQueueMQMD(t *testing.T) {
	start := time.Now().UTC()
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Queue",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	bridgeMessage := message.NewBridgeMessage([]byte(msg))
	bridgeMessage.Header.CorrelID = corr
	bridgeMessage.Header.MsgID = id
	bridgeMessage.SetProperty("count", 11)
	encoded, err := bridgeMessage.Encode()
	require.NoError(t, err)

	err = tbs.NC.Publish("test", encoded)
	require.NoError(t, err)

	mqmd, gmo, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	require.Equal(t, start.Format("20060102"), mqmd.PutDate)
	require.True(t, start.Format("15040500") < mqmd.PutTime)
	require.ElementsMatch(t, id, mqmd.MsgId)
	require.ElementsMatch(t, corr, mqmd.CorrelId)

	require.NotNil(t, gmo)
	require.NotNil(t, gmo.MsgHandle)

	impo := ibmmq.NewMQIMPO()
	pd := ibmmq.NewMQPD()
	impo.Options = ibmmq.MQIMPO_CONVERT_VALUE
	name, value, err := gmo.MsgHandle.InqMP(impo, pd, "count")
	require.NoError(t, err)
	require.Equal(t, "count", name)
	require.Equal(t, int64(11), value.(int64))

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
	require.Equal(t, int64(len(encoded)), connStats.BytesIn)
	require.Equal(t, int64(len(data)), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSimpleSendOnNatsReceiveOnQueueWithTLS(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Queue",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTLSTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	err = tbs.NC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}

func TestWildcardSendRecieveOnQueue(t *testing.T) {
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Queue",
			Subject:        "test.*",
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	err = tbs.NC.Publish("test.a", []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}

func TestSendOnNatsQueueReceiveOnQueue(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Queue",
			Subject:        subject,
			NatsQueue:      "workers",
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	err = tbs.NC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

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
