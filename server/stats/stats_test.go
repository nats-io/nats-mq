package stats

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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

	require.Equal(t, float64(dur.Nanoseconds()), stats.RunningMovingAverage)
	require.Equal(t, int64(1), stats.RequestCount)
	require.Equal(t, uint64(1), stats.RequestTimes.Total)
}

func TestClone(t *testing.T) {
	stats := NewConnectorStats()
	loops := int64(101)
	sizeIn := int64(11)
	sizeOut := int64(13)

	for i := int64(0); i < loops; i++ {
		stats.AddDisconnect()
		stats.AddConnect()
		stats.AddConnect()
		stats.AddMessageIn(sizeIn)
		stats.AddMessageOut(sizeOut)
	}

	dur := 4 * time.Second

	stats.AddRequestTime(dur)

	clone := stats.Clone()

	require.False(t, stats == clone)
	require.False(t, stats.RequestTimes == clone.RequestTimes)
	require.Equal(t, float64(dur.Nanoseconds()), clone.RunningMovingAverage)
	require.Equal(t, int64(1), clone.RequestCount)
	require.Equal(t, uint64(1), clone.RequestTimes.Total)
	require.Equal(t, loops, clone.Disconnects)
	require.Equal(t, 2*loops, clone.Connects)
	require.Equal(t, loops, clone.MessagesIn)
	require.Equal(t, loops, clone.MessagesOut)
	require.Equal(t, loops*sizeIn, clone.BytesIn)
	require.Equal(t, loops*sizeOut, clone.BytesOut)
}
