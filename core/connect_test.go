package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnectToMQTestServer(t *testing.T) {
	mqServer, qMgr, err := StartMQTestServer(5 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, mqServer)
	require.NotNil(t, qMgr)

	time.Sleep(1 * time.Second)

	qMgr.Disc()
	mqServer.Close()
}
