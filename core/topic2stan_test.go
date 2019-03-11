package core

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/message"
	"testing"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/stretchr/testify/require"
)

func TestSimpleSendOnTopicReceiveOnStan(t *testing.T) {
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []ConnectionConfig{
		ConnectionConfig{
			Type:           "Topic2Stan",
			Channel:        channel,
			Topic:          topic,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	received := ""
	done := make(chan bool)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		received = string(msg.Data)
		done <- true
	})
	defer sub.Unsubscribe()

	err = tbs.putMessageOnTopic(topic, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	timer := time.NewTimer(3 * time.Second)
	go func() {
		<-timer.C
		done <- true
	}()

	<-done
	require.Equal(t, msg, received)
}

func TestSendOnTopicReceiveOnStanMQMD(t *testing.T) {
	start := time.Now().UTC()
	channel := "test"
	topic := "dev/"
	msg := "hello world"

	connect := []ConnectionConfig{
		ConnectionConfig{
			Type:           "Topic2Stan",
			Channel:        channel,
			Topic:          topic,
			ExcludeHeaders: false,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	var received []byte
	done := make(chan bool)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		received = msg.Data
		done <- true
	})
	defer sub.Unsubscribe()

	mqmd := ibmmq.NewMQMD()
	err = tbs.putMessageOnTopic(topic, mqmd, []byte(msg))
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
}
