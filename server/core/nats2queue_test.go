package core

import (
	"bytes"
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/server/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnNatsReceiveOnQueue(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		conf.ConnectorConfig{
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
		conf.ConnectorConfig{
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
	encoded, err := bridgeMessage.Encode()
	require.NoError(t, err)

	err = tbs.NC.Publish("test", encoded)
	require.NoError(t, err)

	mqmd, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	require.Equal(t, start.Format("20060102"), mqmd.PutDate)
	require.True(t, start.Format("15040500") < mqmd.PutTime)
	require.ElementsMatch(t, id, mqmd.MsgId)
	require.ElementsMatch(t, corr, mqmd.CorrelId)

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
		conf.ConnectorConfig{
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

	_, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}
