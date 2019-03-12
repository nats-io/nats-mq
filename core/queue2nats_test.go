package core

import (
	"bytes"
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"testing"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnQueueReceiveOnNats(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
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

func TestSendOnQueueReceiveOnNatsMQMD(t *testing.T) {
	start := time.Now().UTC()
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []ConnectorConfig{
		ConnectorConfig{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
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
}
