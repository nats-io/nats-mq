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

func TestSimpleSendOnStanReceiveOnQueue(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Queue",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
	require.Equal(t, int64(len([]byte(msg))), connStats.BytesIn)
	require.Equal(t, int64(len(data)), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSendOnStanReceiveOnQueueMQMD(t *testing.T) {
	start := time.Now().UTC()
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"
	id := bytes.Repeat([]byte{1}, int(ibmmq.MQ_MSG_ID_LENGTH))
	corr := bytes.Repeat([]byte{1}, int(ibmmq.MQ_CORREL_ID_LENGTH))

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Queue",
			Channel:        channel,
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

	err = tbs.SC.Publish("test", encoded)
	require.NoError(t, err)
	err = tbs.SC.Publish("test", encoded)
	require.NoError(t, err)

	mqmd, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	mqmd, data, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	require.Equal(t, start.Format("20060102"), mqmd.PutDate)
	require.True(t, start.Format("15040500") < mqmd.PutTime)
	require.ElementsMatch(t, id, mqmd.MsgId)
	require.ElementsMatch(t, corr, mqmd.CorrelId)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(2), connStats.MessagesIn)
	require.Equal(t, int64(2), connStats.MessagesOut)
	require.Equal(t, int64(2*len(encoded)), connStats.BytesIn)
	require.Equal(t, int64(2*len(data)), connStats.BytesOut)
	require.Equal(t, int64(1), connStats.Connects)
	require.Equal(t, int64(0), connStats.Disconnects)
	require.True(t, connStats.Connected)
}

func TestSimpleSendOnStanReceiveOnQueueWithTLS(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Queue",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTLSTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}

func TestQueueStartAtPosition(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:            "Stan2Queue",
			Channel:         channel,
			Queue:           queue,
			ExcludeHeaders:  true,
			StartAtSequence: 2,
		},
	}

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	// Send 2 messages, should only get 2nd
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	_, _, err = tbs.GetMessageFromQueue(queue, 2000)
	require.Error(t, err)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
}

func TestQueueDeliverLatest(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:            "Stan2Queue",
			Channel:         channel,
			Queue:           queue,
			ExcludeHeaders:  true,
			StartAtSequence: -1,
		},
	}

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	// Send 2 messages, should only get 2nd
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	// Should get the last one
	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	// Should receive 1 message we just sent
	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	_, _, err = tbs.GetMessageFromQueue(queue, 2000)
	require.Error(t, err)

	_, _, err = tbs.GetMessageFromQueue(queue, 2000)
	require.Error(t, err)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(2), connStats.MessagesIn)
	require.Equal(t, int64(2), connStats.MessagesOut)
}

func TestQueueStartAtTime(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	// Send 2 messages, should only get 2nd
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)
	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	time.Sleep(2 * time.Second) // move the time along

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Queue",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: true,
			StartAtTime:    time.Now().Unix(),
		},
	}

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	// Should only get the one we just sent
	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	_, _, err = tbs.GetMessageFromQueue(queue, 2000)
	require.Error(t, err)

	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(1), connStats.MessagesIn)
	require.Equal(t, int64(1), connStats.MessagesOut)
}

func TestQueueDurableSubscriber(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"

	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	connect := []conf.ConnectorConfig{
		{
			Type:           "Stan2Queue",
			Channel:        channel,
			DurableName:    "test_durable",
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte("one"))
	require.NoError(t, err)

	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)

	tbs.StopBridge()

	err = tbs.SC.Publish("test", []byte("two"))
	require.NoError(t, err)

	err = tbs.SC.Publish("test", []byte("three"))
	require.NoError(t, err)

	err = tbs.StartBridge(connect, false)
	require.NoError(t, err)

	// Should only get 2 more, we sent 3 but already got 1
	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	_, _, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	_, _, err = tbs.GetMessageFromQueue(queue, 2000)
	require.Error(t, err)

	// Should have 2 messages since the relaunch
	stats := tbs.Bridge.SafeStats()
	connStats := stats.Connections[0]
	require.Equal(t, int64(2), connStats.MessagesIn)
	require.Equal(t, int64(2), connStats.MessagesOut)
}
