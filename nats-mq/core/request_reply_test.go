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
	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/stretchr/testify/require"
)

func TestSendReceiveOnNATSThruQueue(t *testing.T) {
	subject := "test"
	replyToSubject := "best"
	queue := "DEV.QUEUE.1"
	replyQueue := "DEV.QUEUE.2"
	msg := "hello world"
	response := "goodbye"

	connect := []conf.ConnectorConfig{
		{
			Type:           "NATS2Queue",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
		{
			Type:           "Queue2NATS",
			Subject:        replyToSubject,
			Queue:          replyQueue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan string)

	sub, err := tbs.NC.Subscribe(replyToSubject, func(msg *nats.Msg) {
		done <- string(msg.Data)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = tbs.NC.PublishRequest(subject, replyToSubject, []byte(msg))
	require.NoError(t, err)

	mqmd, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
	require.Equal(t, mqmd.ReplyToQ, replyQueue)

	err = tbs.PutMessageOnQueue(mqmd.ReplyToQ, ibmmq.NewMQMD(), []byte(response))
	require.NoError(t, err)

	timer := time.NewTimer(5 * time.Second)
	go func() {
		<-timer.C
		done <- ""
	}()

	received := <-done
	require.Equal(t, response, received)
}

func TestSendReceiveOnMQThruNATS(t *testing.T) {
	subject := "test"
	replyToSubject := "best"
	queue := "DEV.QUEUE.1"
	replyQueue := "DEV.QUEUE.2"
	msg := "hello world"
	response := "goodbye"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
		{
			Type:           "NATS2Queue",
			Subject:        replyToSubject,
			Queue:          replyQueue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		require.NotNil(t, msg.Reply)
		tbs.NC.Publish(msg.Reply, []byte(response))
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.ReplyToQ = replyQueue
	mqmd.ReplyToQMgr = tbs.GetQueueManagerName()
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(replyQueue, 5000)
	require.NoError(t, err)
	require.Equal(t, response, string(data))
}

func TestSendReceiveOnMQThruNATSHeaderInNotOut(t *testing.T) {
	subject := "test"
	replyToSubject := "best"
	queue := "DEV.QUEUE.1"
	replyQueue := "DEV.QUEUE.2"
	msg := "hello world"
	response := "goodbye"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: false,
		},
		{
			Type:           "NATS2Queue",
			Subject:        replyToSubject,
			Queue:          replyQueue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		_, err := message.DecodeBridgeMessage(msg.Data)
		require.NoError(t, err)
		require.NotNil(t, msg.Reply)
		tbs.NC.Publish(msg.Reply, []byte(response))
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.ReplyToQ = replyQueue
	mqmd.ReplyToQMgr = tbs.GetQueueManagerName()
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(replyQueue, 5000)
	require.NoError(t, err)
	require.Equal(t, response, string(data))
}

func TestSendReceiveOnStanThruQueue(t *testing.T) {
	channel := "test"
	replyToChannel := "best"
	queue := "DEV.QUEUE.1"
	replyQueue := "DEV.QUEUE.2"
	msg := "hello world"
	response := "goodbye"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Queue",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: false,
		},
		{
			Type:           "Queue2Stan",
			Channel:        replyToChannel,
			Queue:          replyQueue,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan string)

	sub, err := tbs.SC.Subscribe(replyToChannel, func(msg *stan.Msg) {
		bridgeMsg, err := message.DecodeBridgeMessage(msg.Data)
		require.NoError(t, err)
		done <- string(bridgeMsg.Body)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	bridgeMsg := message.NewBridgeMessage([]byte(msg))
	bridgeMsg.Header.ReplyToChannel = replyToChannel
	bytes, err := bridgeMsg.Encode()
	require.NoError(t, err)

	err = tbs.SC.Publish(channel, bytes)
	require.NoError(t, err)

	mqmd, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
	require.Equal(t, mqmd.ReplyToQ, replyQueue)

	err = tbs.PutMessageOnQueue(mqmd.ReplyToQ, ibmmq.NewMQMD(), []byte(response))
	require.NoError(t, err)

	timer := time.NewTimer(5 * time.Second)
	go func() {
		<-timer.C
		done <- ""
	}()

	received := <-done
	require.Equal(t, response, received)
}

func TestSendReceiveOnMQThruStan(t *testing.T) {
	channel := "test"
	replyToChannel := "best"
	queue := "DEV.QUEUE.1"
	replyQueue := "DEV.QUEUE.2"
	msg := "hello world"
	response := "goodbye"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Queue2Stan",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: false,
		},
		{
			Type:           "Stan2Queue",
			Channel:        replyToChannel,
			Queue:          replyQueue,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		bridgeMsg, err := message.DecodeBridgeMessage(msg.Data)
		require.NoError(t, err)

		reply := message.NewBridgeMessage([]byte(response))
		replyToChannel := bridgeMsg.Header.ReplyToChannel
		require.NotEmpty(t, replyToChannel)

		encoded, err := reply.Encode()
		require.NoError(t, err)
		tbs.SC.Publish(replyToChannel, encoded)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.ReplyToQ = replyQueue
	mqmd.ReplyToQMgr = tbs.GetQueueManagerName()
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	_, _, data, err := tbs.GetMessageFromQueue(replyQueue, 5000)
	require.NoError(t, err)
	require.Equal(t, response, string(data))
}
