/*
 * Copyright 2012-2019 The NATS Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
