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

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMQTestServer(t *testing.T) {
	mqServer, qMgr, err := StartMQTestServer(5*time.Second, false, 0)
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
	mqServer, qMgr, err := StartMQTestServer(30*time.Second, true, 0)
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
