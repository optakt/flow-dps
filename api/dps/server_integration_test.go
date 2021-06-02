// +build integration

// Copyright 2021 Alvalor S.A.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package dps_test

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

var (
	wantHeight uint64 = 128
	lastHeight uint64 = 256
	testValues        = [][]byte{{0x1, 0x2}, {0x3, 0x4}}
	testPaths         = [][]byte{{0x5, 0x6}, {0x7, 0x8}}
)

func TestMain(m *testing.M) {
	lastCommit, err := flow.ToStateCommitment([]byte("0d339afb6de1aa21b7afbcef3278c8ee"))
	if err != nil {
		println("unable to parse state commitment")
		os.Exit(1)
	}

	mock := mocks.NewState()

	// GetRegisters
	mock.LastState.On("Height").Return(lastHeight)
	mock.RawState.On("WithHeight", wantHeight).Return(mock.RawState)
	for i, path := range testPaths {
		mock.RawState.On("Get", path).Return(testValues[i], nil)
	}

	// GetValues
	mock.LastState.On("Commit").Return(lastCommit)

	testQuery, _ := ledger.NewQuery(ledger.State(lastCommit), nil)
	mock.LedgerState.On("Get", testQuery).Return(testValues, nil)

	server, _ := dps.NewServer(nil)

	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	dps.RegisterAPIServer(s, server)

	go func() {
		if err := s.Serve(lis); err != nil {
			println("unable to setup GRPC api integration tests")
			os.Exit(1)
		}
	}()

	m.Run()

	s.GracefulStop()

	os.Exit(0)
}

func TestServer_GetRegisters(t *testing.T) {
	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	require.NoError(t, err)

	defer conn.Close()

	client := dps.NewAPIClient(conn)

	got, err := client.GetRegisters(ctx, &dps.GetRegistersRequest{
		Height: wantHeight,
		Paths:  testPaths,
	})
	assert.NoError(t, err)

	want := &dps.GetRegistersResponse{
		Height: wantHeight,
		Paths:  testPaths,
		Values: testValues,
	}
	assert.Equal(t, want.Values, got.Values)
	assert.Equal(t, want.Paths, got.Paths)
	assert.Equal(t, want.Height, got.Height)
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}
