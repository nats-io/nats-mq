package core

import (
	"bytes"
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/server/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnQueueReceiveOnStan(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		conf.ConnectorConfig{
			Type:           "Queue2Stan",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan string)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		done <- string(msg.Data)
	})
	defer sub.Unsubscribe()

	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
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

func TestSendOnQueueReceiveOnStanMQMD(t *testing.T) {
	start := time.Now().UTC()
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []conf.ConnectorConfig{
		conf.ConnectorConfig{
			Type:           "Queue2Stan",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan []byte)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		done <- msg.Data
	})
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.CorrelId = corr
	mqmd.MsgId = id
	err = tbs.PutMessageOnQueue(queue, mqmd, []byte(msg))
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
	require.ElementsMatch(t, id, bridgeMessage.Header.MsgID)
	require.ElementsMatch(t, corr, bridgeMessage.Header.CorrelID)

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

func TestSimpleSendOnQueueReceiveOnStanWithTLS(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		conf.ConnectorConfig{
			Type:           "Queue2Stan",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTLSTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	done := make(chan string)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		done <- string(msg.Data)
	})
	defer sub.Unsubscribe()

	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- ""
	}()

	received := <-done
	require.Equal(t, msg, received)
}
