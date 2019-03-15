package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMQTestServer(t *testing.T) {
	mqServer, qMgr, err := StartMQTestServer(5*time.Second, false)
	defer func() {
		if qMgr != nil {
			qMgr.Disc()
		}
		if mqServer != nil {
			mqServer.Close()
		}
	}()
	require.NoError(t, err)
	require.NotNil(t, mqServer)
	require.NotNil(t, qMgr)

	time.Sleep(1 * time.Second)
}

func TestMQTestServerWithTLS(t *testing.T) {
	mqServer, qMgr, err := StartMQTestServer(30*time.Second, true)
	defer func() {
		if qMgr != nil {
			qMgr.Disc()
		}
		if mqServer != nil {
			mqServer.Close()
		}
	}()
	require.NoError(t, err)
	require.NotNil(t, mqServer)
	require.NotNil(t, qMgr)

	time.Sleep(1 * time.Second)
}
