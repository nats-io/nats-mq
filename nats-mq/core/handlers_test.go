package core

import (
	"testing"
	"time"

	"github.com/ibm-messaging/mq-golang/ibmmq"
	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/stretchr/testify/require"
)

func TestMQReconnect(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Queue2NATS",
			Subject:        subject,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	sub, err := tbs.NC.SubscribeSync(subject)
	defer sub.Unsubscribe()

	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	received, err := sub.NextMsg(5 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, received)
	require.Equal(t, msg, string(received.Data))

	err = tbs.RestartMQ(false)
	require.NoError(t, err)

	// Wait up to 15s for the bridge to get reconnected
	start := time.Now()
	for time.Now().Sub(start) < time.Second*15 && tbs.Bridge.checkReconnecting() {
		time.Sleep(250 * time.Millisecond)
	}

	require.False(t, tbs.Bridge.checkReconnecting())

	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	// sub should have auto reconnected
	received, err = sub.NextMsg(5 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, received)
	require.Equal(t, msg, string(received.Data))
}

func TestNATSReconnect(t *testing.T) {
	subject := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
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

	_, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	err = tbs.RestartNATS(false)
	require.NoError(t, err)

	// Wait up to 15s for the bridge to get reconnected
	start := time.Now()
	for time.Now().Sub(start) < time.Second*15 && tbs.Bridge.checkReconnecting() {
		time.Sleep(250 * time.Millisecond)
	}

	require.False(t, tbs.Bridge.checkReconnecting())

	err = tbs.NC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, _, data, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}

func TestStanReconnect(t *testing.T) {
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

	_, _, data, err := tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))

	err = tbs.RestartNATS(false)
	require.NoError(t, err)

	// Wait up to 15s for the bridge to get reconnected
	start := time.Now()
	for time.Now().Sub(start) < time.Second*15 && tbs.Bridge.checkReconnecting() {
		time.Sleep(250 * time.Millisecond)
	}

	require.False(t, tbs.Bridge.checkReconnecting())

	err = tbs.SC.Publish("test", []byte(msg))
	require.NoError(t, err)

	_, _, data, err = tbs.GetMessageFromQueue(queue, 5000)
	require.NoError(t, err)
	require.Equal(t, msg, string(data))
}

/*
func TestStanPubFailure(t *testing.T) {
	channel := "test"
	queue := "DEV.QUEUE.1"
	msg := "hello world"

	connect := []conf.ConnectorConfig{
		{
			Type:           "Queue2Stan",
			Channel:        channel,
			Queue:          queue,
			ExcludeHeaders: true,
		},
	}

	tbs, err := StartTestEnvironment(connect)
	require.NoError(t, err)
	defer tbs.Close()

	wg := sync.WaitGroup{}
	wg.Add(3)

	sub, err := tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		wg.Done()
	})
	defer sub.Unsubscribe()

	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	err = tbs.StopNATS()
	require.NoError(t, err)

	// Queue up 2 more
	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	err = tbs.PutMessageOnQueue(queue, ibmmq.NewMQMD(), []byte(msg))
	require.NoError(t, err)

	err = tbs.RestartNATS(false)
	require.NoError(t, err)

	// Resubscribe, since we lost the connection
	sub, err = tbs.SC.Subscribe(channel, func(msg *stan.Msg) {
		wg.Done()
	})
	defer sub.Unsubscribe()

	// Wait up to 15s for the bridge to get reconnected
	start := time.Now()
	for time.Now().Sub(start) < time.Second*15 && tbs.Bridge.checkReconnecting() {
		time.Sleep(250 * time.Millisecond)
	}

	require.False(t, tbs.Bridge.checkReconnecting())

	timedOut := false

	// Don't let wg wait forever
	timer := newReconnectTimer()
	go func() {
		select {
		case timedOut = <-timer.After(20 * time.Second):
			if !timedOut {
				return
			}
		}

		wg.Done()
		wg.Done()
		wg.Done()
	}()

	wg.Wait()
	timer.Cancel()
	require.False(t, timedOut)
}
*/
