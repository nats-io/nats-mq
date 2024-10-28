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
 *
 */

package core

import (
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnStanReceiveOnTopic(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Topic",
			Channel:        channel,
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

	err = tbs.SC.Publish("test", []byte(msg))
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
	require.Equal(t, int64(datalen), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSendOnStanReceiveOnTopicMQMD(t *testing.T) {
	var topicObject ibmmq.MQObject
	start := time.Now().UTC()
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Topic",
			Channel:        channel,
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
	encoded, err := bridgeMessage.Encode()
	require.NoError(t, err)

	err = tbs.SC.Publish("test", encoded)
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
	require.Equal(t, int64(len(encoded)), connStats.BytesIn)
	require.Equal(t, int64(datalen), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSimpleSendOnStanReceiveOnTopicTLS(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Topic",
			Channel:        channel,
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

	err = tbs.SC.Publish("test", []byte(msg))
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

func TestTopicStartAtPosition(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:            "Stan2Topic",
			Channel:         channel,
			Topic:           topic,
			ExcludeHeaders:  true,
			StartAtSequence: 2,
		},
	}

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	// Send 2 messages, should only get 2nd
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.Error(t, err)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
}

func TestTopicDeliverLatest(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"

	connect := []conf.ConnectorConfig{
		{
			Type:            "Stan2Topic",
			Channel:         channel,
			Topic:           topic,
			ExcludeHeaders:  true,
			StartAtSequence: -1,
		},
	}

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 4 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	// Send 2 messages, should only get 2nd
	err = tbs.SC.Publish("test", []byte("one"))
	require.NoError(t, err)
	err = tbs.SC.Publish("test", []byte("two"))
	require.NoError(t, err)

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	// Should get the last one we sent before bridge started
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte("three"))
	require.NoError(t, err)

	// Should receive 1 message we just sent
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.Error(t, err)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(2), connStats.MessagesIn)
	require.Equal(t, int64(2), connStats.MessagesOut)
}

func TestTopicStartAtTime(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	// Send 2 messages, should only get 2nd
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // move the time along

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Topic",
			Channel:        channel,
			Topic:          topic,
			ExcludeHeaders: true,
			StartAtTime:    time.Now().Unix(),
		},
	}

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	// Should only get the one we just sent
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.Error(t, err)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
}

func TestTopicDurableSubscriber(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE | ibmmq.MQSO_NON_DURABLE | ibmmq.MQSO_MANAGED
	mqsd.ObjectString = topic
	sub, err := tbs.QMgr.Sub(mqsd, &topicObject)
	require.NoError(t, err)
	defer sub.Close(0)

	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds
	buffer := make([]byte, 1024)

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Topic",
			Channel:        channel,
			Topic:          topic,
			DurableName:    "test_durable",
			ExcludeHeaders: true,
		},
	}

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte("one"))
	require.NoError(t, err)

	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)

	tbs.StopBridge()

	err = tbs.SC.Publish("test", []byte("two"))
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte("three"))
	require.NoError(t, err)

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	// Should only get 2 more, we sent 3 but already got 1
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.NoError(t, err)
	_, err = topicObject.Get(ibmmq.NewMQMD(), gmo, buffer)
	require.Error(t, err)

	// Should have 2 messages since the relaunch
	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(2), connStats.MessagesIn)
	require.Equal(t, int64(2), connStats.MessagesOut)
}
