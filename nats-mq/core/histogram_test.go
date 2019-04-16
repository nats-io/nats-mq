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
 */

package core

// Based on https://github.com/VividCortex/gohistogram MIT license

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func approx(x, y float64) bool {
	return math.Abs(x-y) < 0.2
}

func TestHistogram(t *testing.T) {
	h := NewHistogram(160)
	for _, val := range testData {
		h.Add(float64(val))
	}

	firstQ := h.Quantile(0.25)
	median := h.Quantile(0.5)
	thirdQ := h.Quantile(0.75)

	require.Equal(t, float64(14999), h.Count())
	require.True(t, approx(firstQ, 14))
	require.True(t, approx(median, 18))
	require.True(t, approx(thirdQ, 22))

	require.True(t, approx(h.Mean(), 20.7))

	h.Scale(0.5)
	median = h.Quantile(0.5)
	require.True(t, approx(median, 9))
}

func TestHistogramTrim(t *testing.T) {
	h := NewHistogram(10)
	for _, val := range testData {
		h.Add(float64(val))
	}

	require.Equal(t, 10, len(h.Bins))
}
