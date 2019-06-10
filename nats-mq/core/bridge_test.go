// Copyright 2012-2019 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package core

import (
	"testing"

	"github.com/nats-io/nats-mq/nats-mq/conf"
	"github.com/nats-io/nuid"
	stan "github.com/nats-io/stan.go"
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

	// Use a bogus server because we don't want to default to one on the test machine
	config.NATS = conf.NATSConfig{
		Servers:        []string{"nats://abc:321"},
		ConnectTimeout: 2000,
		ReconnectWait:  2000,
		MaxReconnects:  5,
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
