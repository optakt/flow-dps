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

package rest_test

import (
	"testing"

	exec "github.com/onflow/flow-go/engine/execution/state"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/api/rest"
	"github.com/stretchr/testify/assert"
)

func TestEncodeKey(t *testing.T) {
	tests := map[string]struct {
		key  ledger.Key
		want string
	}{
		"nominal case": {
			key:  exec.RegisterIDToKey(flow.NewRegisterID("testOwner", "testController", "testKey")),
			want: "0.746573744f776e6572,1.74657374436f6e74726f6c6c6572,2.746573744b6579",
		},
		"empty key parts": {
			key:  exec.RegisterIDToKey(flow.NewRegisterID("", "", "testKey")),
			want: "0.,1.,2.746573744b6579",
		},
		"empty key": {
			key:  ledger.Key{},
			want: "",
		},
	}

	for desc, test := range tests {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			got := rest.EncodeKey(test.key)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestEncodeKeys(t *testing.T) {
	tests := map[string]struct {
		keys  []ledger.Key
		want string
	}{
		"nominal case": {
			keys:  []ledger.Key{
				exec.RegisterIDToKey(flow.NewRegisterID("testOwner", "testController", "testKey")),
				exec.RegisterIDToKey(flow.NewRegisterID("testOwner2", "testController2", "testKey2")),
			},
			want: "0.746573744f776e6572,1.74657374436f6e74726f6c6c6572,2.746573744b6579:0.746573744f776e657232,1.74657374436f6e74726f6c6c657232,2.746573744b657932",
		},
		"empty key parts": {
			keys:  []ledger.Key{
				exec.RegisterIDToKey(flow.NewRegisterID("", "", "testKey")),
				exec.RegisterIDToKey(flow.NewRegisterID("", "", "testKey2")),
			},
			want: "0.,1.,2.746573744b6579:0.,1.,2.746573744b657932",
		},
		"empty keys": {
			keys:  []ledger.Key{{}, {}},
			want: ":",
		},
		"no keys": {
			keys:  []ledger.Key{},
			want: "",
		},
	}

	for desc, test := range tests {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			got := rest.EncodeKeys(test.keys)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestDecodeKey(t *testing.T) {
	tests := map[string]struct {
		key     string
		want    ledger.Key
		wantErr assert.ErrorAssertionFunc
	}{
		"nominal case": {
			key: "0.746573744f776e6572,1.74657374436f6e74726f6c6c6572,2.746573744b6579",

			want:    exec.RegisterIDToKey(flow.NewRegisterID("testOwner", "testController", "testKey")),
			wantErr: assert.NoError,
		},
		"empty key parts": {
			key: "0.,1.,2.746573744b6579",

			want:    exec.RegisterIDToKey(flow.NewRegisterID("", "", "testKey")),
			wantErr: assert.NoError,
		},
		"empty key": {
			key: "",

			wantErr: assert.Error,
		},
	}

	for desc, test := range tests {
		test := test
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			got, err := rest.DecodeKey(test.key)
			test.wantErr(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}
