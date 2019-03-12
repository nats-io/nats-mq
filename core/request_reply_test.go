package core

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"testing"
	"time"

	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/stretchr/testify/require"
)

func TestSendReceiveOnNATSThruQueue(t *testing.T) {
	subject := "test"
	replyToSubject := "best"
	queue := "DEV.QUEUE.1"
	replyQueue := "DEV.QUEUE.2"
	msg := "hello world"
	response := "goodbye"

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "NATS2Queue",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
		ConnectorConfig{
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
	defer sub.Unsubscribe()

	err = tbs.NC.PublishRequest(subject, replyToSubject, []byte(msg))
	require.NoError(t, err)

	mqmd, data, err := tbs.GetMessageFromQueue(queue, 5000)
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

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
		ConnectorConfig{
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
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.ReplyToQ = replyQueue
	mqmd.ReplyToQMgr = tbs.GetQueueManagerName()
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	_, data, err := tbs.GetMessageFromQueue(replyQueue, 5000)
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

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: false,
		},
		ConnectorConfig{
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
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.ReplyToQ = replyQueue
	mqmd.ReplyToQMgr = tbs.GetQueueManagerName()
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	_, data, err := tbs.GetMessageFromQueue(replyQueue, 5000)
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

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "Stan2Queue",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: false,
		},
		ConnectorConfig{
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
	defer sub.Unsubscribe()

	bridgeMsg := message.NewBridgeMessage([]byte(msg))
	bridgeMsg.Header.ReplyToChannel = replyToChannel
	bytes, err := bridgeMsg.Encode()
	require.NoError(t, err)

	err = tbs.SC.Publish(channel, bytes)
	require.NoError(t, err)

	mqmd, data, err := tbs.GetMessageFromQueue(queue, 5000)
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

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "Queue2Stan",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: false,
		},
		ConnectorConfig{
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
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.ReplyToQ = replyQueue
	mqmd.ReplyToQMgr = tbs.GetQueueManagerName()
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	_, data, err := tbs.GetMessageFromQueue(replyQueue, 5000)
	require.NoError(t, err)
	require.Equal(t, response, string(data))
}
