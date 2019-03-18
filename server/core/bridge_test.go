package core

import (
	"testing"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats-mq/server/conf"
	"github.com/nats-io/nuid"
	"github.com/stretchr/testify/require"
)

func TestStartBridgeNoNats(t *testing.T) {
	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	config := conf.DefaultBridgeConfig()
	config.Monitoring = conf.MonitoringConfig{
		HTTPPort: -1,
	}

	bridge := NewBridgeServer()
	err = bridge.LoadConfig(config)
	require.NoError(t, err)

	err = bridge.Start()
	require.Error(t, err)
}

func TestStartBridgeNATSOnly(t *testing.T) {
	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	config := conf.DefaultBridgeConfig()
	config.Monitoring = conf.MonitoringConfig{
		HTTPPort: -1,
	}
	config.NATS = conf.NATSConfig{
		Servers:        []string{tbs.natsURL},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
	}

	bridge := NewBridgeServer()
	err = bridge.LoadConfig(config)
	require.NoError(t, err)

	err = bridge.Start()
	require.NoError(t, err)

	bridge.Stop()
}

func TestStartBridgeNATSAndStan(t *testing.T) {
	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	config := conf.DefaultBridgeConfig()
	config.Monitoring = conf.MonitoringConfig{
		HTTPPort: -1,
	}
	config.NATS = conf.NATSConfig{
		Servers:        []string{tbs.natsURL},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
	}
	config.STAN = conf.NATSStreamingConfig{
		ClusterID:          tbs.clusterName,
		ClientID:           nuid.Next(),
		PubAckWait:         5000,
		DiscoverPrefix:     stan.DefaultDiscoverPrefix,
		MaxPubAcksInflight: stan.DefaultMaxPubAcksInflight,
		ConnectWait:        2000,
	}

	bridge := NewBridgeServer()
	err = bridge.LoadConfig(config)
	require.NoError(t, err)

	err = bridge.Start()
	require.NoError(t, err)

	bridge.Stop()
}

func TestCantHaveHTTPAndHTTPSMonitoring(t *testing.T) {
	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	config := conf.DefaultBridgeConfig()
	config.Monitoring = conf.MonitoringConfig{
		HTTPPort:  9191,
		HTTPSPort: 9192,
	}
	config.NATS = conf.NATSConfig{
		Servers:        []string{tbs.natsURL},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
	}

	bridge := NewBridgeServer()
	err = bridge.LoadConfig(config)
	require.NoError(t, err)

	err = bridge.Start()
	require.Error(t, err)
	bridge.Stop()
}

func TestMonitoringDisabled(t *testing.T) {
	tbs, err := StartTestEnvironmentInfrastructure(false)
	require.NoError(t, err)
	defer tbs.Close()

	config := conf.DefaultBridgeConfig()
	config.Monitoring = conf.MonitoringConfig{
		HTTPPort:  0,
		HTTPSPort: 0,
	}
	config.NATS = conf.NATSConfig{
		Servers:        []string{tbs.natsURL},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
	}

	bridge := NewBridgeServer()
	err = bridge.LoadConfig(config)
	require.NoError(t, err)

	err = bridge.Start()
	require.NoError(t, err)

	require.Equal(t, "", bridge.GetMonitoringRootURL())

	bridge.Stop()
}
