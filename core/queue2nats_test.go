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

	connect := []ConnectionConfig{
		ConnectionConfig{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	received := ""
	done := make(chan bool)

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		received = string(msg.Data)
		done <- true
	})
	defer sub.Unsubscribe()

	err = tbs.putMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- true
	}()

	<-done
	require.Equal(t, msg, received)
}

func TestSendOnQueueReceiveOnNatsMQMD(t *testing.T) {
	start := time.Now().UTC()
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []ConnectionConfig{
		ConnectionConfig{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	var received []byte
	done := make(chan bool)

	sub, err := tbs.NC.Subscribe(subject, func(msg *nats.Msg) {
		received = msg.Data
		done <- true
	})
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	mqmd.CorrelId = corr
	mqmd.MsgId = id
	err = tbs.putMessageOnQueue(queue, mqmd, []byte(msg))
	require.NoError(t, err)

	// don't wait forever
	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- true
	}()

	<-done

	require.True(t, len(received) > 0)

	bridgeMessage, err := message.DecodeBridgeMessage(received)
	require.NoError(t, err)

	require.Equal(t, msg, string(bridgeMessage.Body))
	require.Equal(t, start.Format("20060102"), bridgeMessage.Header.PutDate)
	require.True(t, start.Format("15040500") < bridgeMessage.Header.PutTime)
	require.ElementsMatch(t, id, bridgeMessage.Header.MsgID)
	require.ElementsMatch(t, corr, bridgeMessage.Header.CorrelID)
}
