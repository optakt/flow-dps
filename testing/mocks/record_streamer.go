// Copyright 2021 Optakt Labs OÜ
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

package mocks

import (
	"testing"

	"github.com/onflow/flow-go/engine/execution/ingestion/uploader"
)

type RecordStreamer struct {
	NextFunc func() (*uploader.BlockData, error)
}

func BaselineRecordStreamer(t *testing.T) *RecordStreamer {
	t.Helper()

	r := RecordStreamer{
		NextFunc: func() (*uploader.BlockData, error) {
			return GenericRecord(), nil
		},
	}

	return &r
}

func (r *RecordStreamer) Next() (*uploader.BlockData, error) {
	return r.NextFunc()
}
