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

package gcp

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"

	"github.com/onflow/flow-go/model/flow"
)

type Reader struct {
	bucket *storage.BucketHandle

	// the index of the next segment to retrieve.
	index int
}

// NewReader creates a Reader instance that has access to a GCP bucket.
func NewReader(bucket *storage.BucketHandle) *Reader {
	d := Reader{
		bucket: bucket,
	}

	return &d
}

// Read reads and returns the contents of the next available segment.
func (d *Reader) Read(blockID flow.Identifier) ([]byte, error) {
	block := d.bucket.Object(fmt.Sprintf("%x.cbor", blockID))
	r, err := block.NewReader(context.TODO()) // FIXME: Use a proper context
	if err != nil {
		return nil, fmt.Errorf("could not read bucket item: %w", err)
	}
	defer r.Close()

	return io.ReadAll(r)
}
