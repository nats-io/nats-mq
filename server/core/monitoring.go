package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// HTTP endpoints
const (
	RootPath    = "/"
	VarzPath    = "/varz"
	HealthzPath = "/healthz"
)

// startMonitoring starts the HTTP or HTTPs server if needed.
// expects the lock to be held
func (bridge *BridgeServer) startMonitoring() error {
	config := bridge.config.Monitoring

	if config.HTTPPort != 0 && config.HTTPSPort != 0 {
		return fmt.Errorf("can't specify both HTTP (%v) and HTTPs (%v) ports", config.HTTPPort, config.HTTPSPort)
	}

	if config.HTTPPort == 0 && config.HTTPSPort == 0 {
		bridge.logger.Noticef("monitoring is disabled")
		return nil
	}

	secure := false

	if config.HTTPSPort != 0 {
		if config.TLS.Cert == "" || config.TLS.Key == "" {
			return fmt.Errorf("TLS cert and key required for HTTPS")
		}
		secure = true
	}

	// Used to track HTTP requests
	bridge.httpReqStats = map[string]int64{
		RootPath:    0,
		VarzPath:    0,
		HealthzPath: 0,
	}

	var (
		hp           string
		err          error
		httpListener net.Listener
		port         int
	)

	monitorProtocol := "http"

	if secure {
		monitorProtocol += "s"
		port = config.HTTPSPort
		if port == -1 {
			port = 0
		}
		hp = net.JoinHostPort(config.HTTPHost, strconv.Itoa(port))

		cer, err := tls.LoadX509KeyPair(config.TLS.Cert, config.TLS.Key)
		if err != nil {
			return err
		}

		config := &tls.Config{Certificates: []tls.Certificate{cer}}
		config.ClientAuth = tls.NoClientCert
		httpListener, err = tls.Listen("tcp", hp, config)

	} else {
		port = config.HTTPPort
		if port == -1 {
			port = 0
		}
		hp = net.JoinHostPort(config.HTTPHost, strconv.Itoa(port))
		httpListener, err = net.Listen("tcp", hp)
	}

	if err != nil {
		return fmt.Errorf("can't listen to the monitor port: %v", err)
	}

	bridge.logger.Noticef("starting %s monitor on %s", monitorProtocol,
		net.JoinHostPort(config.HTTPHost, strconv.Itoa(httpListener.Addr().(*net.TCPAddr).Port)))

	mhp := net.JoinHostPort(config.HTTPHost, strconv.Itoa(httpListener.Addr().(*net.TCPAddr).Port))
	if config.HTTPHost == "" {
		mhp = "localhost" + mhp
	}
	bridge.monitoringURL = fmt.Sprintf("%s://%s/", monitorProtocol, mhp)

	mux := http.NewServeMux()

	mux.HandleFunc(RootPath, bridge.HandleRoot)
	mux.HandleFunc(VarzPath, bridge.HandleVarz)
	mux.HandleFunc(HealthzPath, bridge.HandleHealthz)

	// Do not set a WriteTimeout because it could cause cURL/browser
	// to return empty response or unable to display page if the
	// server needs more time to build the response.
	srv := &http.Server{
		Addr:           hp,
		Handler:        mux,
		MaxHeaderBytes: 1 << 20,
	}

	bridge.httpListener = httpListener
	bridge.httpHandler = mux
	bridge.monitoringServer = srv

	go func() {
		srv.Serve(httpListener)
		srv.Handler = nil
	}()

	return nil
}

// StopMonitoring shuts down the http server used for monitoring
// expects the lock to be held
func (bridge *BridgeServer) StopMonitoring() error {
	bridge.logger.Tracef("stopping monitoring")
	if bridge.httpHandler != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
		defer cancel()

		if err := bridge.monitoringServer.Shutdown(ctx); err != nil {
			return err
		}

		bridge.monitoringServer = nil
		bridge.httpHandler = nil
	}

	if bridge.httpListener != nil {
		bridge.httpListener.Close() // ignore the error
		bridge.httpListener = nil
	}
	bridge.logger.Noticef("http monitoring stopped")

	return nil
}

// HandleRoot will show basic info and links to others handlers.
func (bridge *BridgeServer) HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	bridge.Lock()
	bridge.httpReqStats[RootPath]++
	bridge.Unlock()
	fmt.Fprintf(w, `<html lang="en">
   <head>
    <link rel="shortcut icon" href="http://nats.io/img/favicon.ico">
    <style type="text/css">
      body { font-family: "Century Gothic", CenturyGothic, AppleGothic, sans-serif; font-size: 22; }
      a { margin-left: 32px; }
    </style>
  </head>
  <body>
    <img src="http://nats.io/img/logo.png" alt="NATS">
    <br/>
		<a href=/varz>varz</a><br/>
		<a href=/healthz>healthz</a><br/>
    <br/>
  </body>
</html>`)
}

// HandleVarz returns statistics about the server.
func (bridge *BridgeServer) HandleVarz(w http.ResponseWriter, r *http.Request) {
	bridge.Lock()
	bridge.httpReqStats[VarzPath]++
	stats := bridge.stats()
	bridge.Unlock()

	varzJSON, err := json.Marshal(stats)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(varzJSON)
}

// HandleHealthz returns status 200.
func (bridge *BridgeServer) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	bridge.Lock()
	bridge.httpReqStats[HealthzPath]++
	bridge.Unlock()
	w.WriteHeader(http.StatusOK)
}

// stats calculates the stats for the server and connectors
// assumes that the lock is held by the caller
func (bridge *BridgeServer) stats() BridgeStats {
	stats := BridgeStats{}

	now := time.Now()
	stats.StartTime = bridge.startTime.Unix()
	stats.UpTime = now.Sub(bridge.startTime).String()
	stats.ServerTime = now.Unix()

	for _, connector := range bridge.connectors {
		stats.Connections = append(stats.Connections, connector.Stats())
	}

	stats.HTTPRequests = map[string]int64{}

	for k, v := range bridge.httpReqStats {
		stats.HTTPRequests[k] = int64(v)
	}

	return stats
}

// SafeStats grabs the lock then calls stats(), useful for tests
func (bridge *BridgeServer) SafeStats() BridgeStats {
	bridge.Lock()
	bridge.Unlock()
	return bridge.stats()
}

// GetMonitoringRootURL returns the protocol://host:port for the monitoring server, useful for testing
func (bridge *BridgeServer) GetMonitoringRootURL() string {
	bridge.Lock()
	bridge.Unlock()
	return bridge.monitoringURL
}
