package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMessageCounts(t *testing.T) {
	stats := NewConnectorStats()
	loops := int64(101)
	sizeIn := int64(11)
	sizeOut := int64(13)

	for i := int64(0); i < loops; i++ {
		stats.AddMessageIn(sizeIn)
		stats.AddMessageOut(sizeOut)
	}

	require.Equal(t, loops, stats.MessagesIn)
	require.Equal(t, loops, stats.MessagesOut)
	require.Equal(t, loops*sizeIn, stats.BytesIn)
	require.Equal(t, loops*sizeOut, stats.BytesOut)
}

func TestConnectDisconnectCounts(t *testing.T) {
	stats := NewConnectorStats()
	loops := int64(101)

	for i := int64(0); i < loops; i++ {
		stats.AddDisconnect()
		stats.AddConnect()
		stats.AddConnect()
	}

	require.Equal(t, loops, stats.Disconnects)
	require.Equal(t, 2*loops, stats.Connects)
	require.Equal(t, int64(0), stats.BytesIn)
	require.Equal(t, int64(0), stats.BytesOut)
	require.Equal(t, int64(0), stats.MessagesIn)
	require.Equal(t, int64(0), stats.MessagesOut)
}

func TestRequestTimes(t *testing.T) {
	stats := NewConnectorStats()

	dur := 4 * time.Second

	stats.AddRequestTime(dur)

	require.Equal(t, float64(dur.Nanoseconds()), stats.MovingAverage)
	require.Equal(t, int64(1), stats.RequestCount)
}
