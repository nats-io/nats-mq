package core

import (
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"github.com/nats-io/nats-mq/server/conf"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnStanReceiveOnTopic(t *testing.T) {
	var topicObject ibmmq.MQObject
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		conf.ConnectorConfig{
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
		conf.ConnectorConfig{
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
		conf.ConnectorConfig{
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
