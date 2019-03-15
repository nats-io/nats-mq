package core

import (
	"time"
)

// BridgeStats wraps the current status of the bridge and all of its connectors
type BridgeStats struct {
	StartTime    int64            `json:"start_time"`
	ServerTime   int64            `json:"current_time"`
	UpTime       string           `json:"uptime"`
	Connections  []ConnectorStats `json:"connectors"`
	HTTPRequests map[string]int64 `json:"http_requests"`
}

// ConnectorStats captures the statistics for a single connector
type ConnectorStats struct {
	Connected     bool    `json:"connected"`
	Connects      int64   `json:"connects"`
	Disconnects   int64   `json:"disconnects"`
	Name          string  `json:"name"`
	BytesIn       int64   `json:"bytes_in"`
	BytesOut      int64   `json:"bytes_out"`
	MessagesIn    int64   `json:"msg_in"`
	MessagesOut   int64   `json:"msg_out"`
	RequestCount  int64   `json:"count"`
	MovingAverage float64 `json:"rma"`
	Quintile50    float64 `json:"q50"`
	Quintile75    float64 `json:"q75"`
	Quintile90    float64 `json:"q90"`
	Quintile95    float64 `json:"q95"`
	histogram     *Histogram
}

// NewConnectorStats creates an empty stats, and initializes the request time histogram
func NewConnectorStats() ConnectorStats {
	return ConnectorStats{
		histogram: NewHistogram(60),
	}
}

// AddMessageIn updates the messages in and bytes in fields
func (stats *ConnectorStats) AddMessageIn(bytes int64) {
	stats.MessagesIn++
	stats.BytesIn += bytes
}

// AddMessageOut updates the messages out and bytes out fields
func (stats *ConnectorStats) AddMessageOut(bytes int64) {
	stats.MessagesOut++
	stats.BytesOut += bytes
}

// AddDisconnect updates the disconnects field
func (stats *ConnectorStats) AddDisconnect() {
	stats.Disconnects++
	stats.Connected = false
}

// AddConnect updates the reconnects field
func (stats *ConnectorStats) AddConnect() {
	stats.Connects++
	stats.Connected = true
}

// AddRequestTime register a time, updating the request count, RMA and histogram
// For information on the running moving average, see https://en.wikipedia.org/wiki/Moving_average
func (stats *ConnectorStats) AddRequestTime(reqTime time.Duration) {
	reqns := float64(reqTime.Nanoseconds())
	stats.RequestCount++
	stats.MovingAverage = ((float64(stats.RequestCount-1) * stats.MovingAverage) + reqns) / float64(stats.RequestCount)
	stats.histogram.Add(reqns)
}

// UpdateQuintiles updates the quantile fields, these are not updated on each request
// to reduce the cost of tracking statistics
func (stats *ConnectorStats) UpdateQuintiles() {
	stats.Quintile50 = stats.histogram.Quantile(0.5)
	stats.Quintile75 = stats.histogram.Quantile(0.75)
	stats.Quintile90 = stats.histogram.Quantile(0.9)
	stats.Quintile95 = stats.histogram.Quantile(0.95)
}
