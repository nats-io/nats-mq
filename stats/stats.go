package stats

import (
	"time"
)

// BridgeStats wraps the current status of the bridge and all of its connectors
type BridgeStats struct {
	StartTime   int64            `json:"start_time"`
	ServerTime  int64            `json:"current_time"`
	UpTime      string           `json:"uptime"`
	Connections []ConnectorStats `json:"connectors"`
}

// ConnectorStats captures the statistics for a single connector
type ConnectorStats struct {
	Connected            bool       `json:"connected"`
	Disconnects          int64      `json:"disconnecnts"`
	Connects             int64      `json:"connects"`
	BytesIn              int64      `json:"bytes_in"`
	BytesOut             int64      `json:"bytes_out"`
	MessagesIn           int64      `json:"msg_in"`
	MessagesOut          int64      `json:"msg_out"`
	RequestTimes         *Histogram `json:"histogram"`
	RunningMovingAverage float64    `json:"rma"`
	RequestCount         int64      `json:"count"`
}

// NewConnectorStats creates an empty stats, and initializes the request time histogram
func NewConnectorStats() *ConnectorStats {
	stats := &ConnectorStats{}
	stats.RequestTimes = NewHistogram(40)
	return stats
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
}

// AddConnect updates the reconnects field
func (stats *ConnectorStats) AddConnect() {
	stats.Connects++
}

// AddRequestTime register a time, updating the request count, RMA and histogram
func (stats *ConnectorStats) AddRequestTime(reqTime time.Duration) {
	reqns := float64(reqTime.Nanoseconds())
	stats.RequestCount++
	stats.RunningMovingAverage = ((float64(stats.RequestCount-1) * stats.RunningMovingAverage) + reqns) / float64(stats.RequestCount)
	stats.RequestTimes.Add(reqns)
}

// Clone the stats in a thread safe way
func (stats *ConnectorStats) Clone() *ConnectorStats {
	histogram := *stats.RequestTimes
	other := *stats
	other.RequestTimes = &histogram
	return &other
}
