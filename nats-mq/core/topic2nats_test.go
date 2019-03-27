package core

import (
	"bytes"
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	nats "github.com/nats-io/go-nats"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnTopicReceiveOnNats(t *testing.T) {
	subject := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Topic2NATS",
			Subject:        subject,
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan string)

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		done <- string(msg.Data)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = tbs.PutMessageOnTopic(topic, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- ""
	}()

	received := <-done
	require.Equal(t, msg, received)

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

func TestSendOnTopicReceiveOnNatsMQMD(t *testing.T) {
	start := time.Now().UTC()
	subject := "test"
	topic := "dev/"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []conf.ConnectorConfig{
		{
			Type:           "Topic2NATS",
			Subject:        subject,
			Topic:          topic,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan []byte)

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		done <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.CorrelId = corr
	mqmd.MsgId = id
	err = tbs.PutMessageOnTopic(topic, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	// don't wait forever
	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- []byte{}
	}()

	received := <-done

	require.True(t, len(received) > 0)

	bridgeMessage, err := message.DecodeBridgeMessage(received)
	require.NoError(t, err)

	require.Equal(t, msg, string(bridgeMessage.Body))
	require.Equal(t, start.Format("20060102"), bridgeMessage.Header.PutDate)
	require.True(t, start.Format("15040500") < bridgeMessage.Header.PutTime)

	// TODO looks like topics generate these, perhaps it is a setting
	//require.ElementsMatch(t, id, bridgeMessage.Header.MsgID)
	//require.ElementsMatch(t, corr, bridgeMessage.Header.CorrelID)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
	require.Equal(t, int64(len([]byte(msg))), connStats.BytesIn)
	require.Equal(t, int64(len(received)), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSimpleSendOnTopicReceiveOnNatsWithTLS(t *testing.T) {
	subject := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Topic2NATS",
			Subject:        subject,
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTLSTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan string)

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		done <- string(msg.Data)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	err = tbs.PutMessageOnTopic(topic, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- ""
	}()

	received := <-done
	require.Equal(t, msg, received)
}
