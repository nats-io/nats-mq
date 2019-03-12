package core

import (
	"bytes"
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnNatsReceiveOnQueue(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []ConnectorConfig{
		ConnectorConfig{
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

	_, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}

func TestSendOnNATSReceiveOnQueueMQMD(t *testing.T) {
	start := time.Now().UTC()
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []ConnectorConfig{
		ConnectorConfig{
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
	data, err := bridgeMessage.Encode()
	require.NoError(t, err)

	err = tbs.NC.Publish("test", data)
	require.NoError(t, err)

	mqmd, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	require.Equal(t, start.Format("20060102"), mqmd.PutDate)
	require.True(t, start.Format("15040500") < mqmd.PutTime)
	require.ElementsMatch(t, id, mqmd.MsgId)
	require.ElementsMatch(t, corr, mqmd.CorrelId)
}
