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

import "testing"

type Codec struct {
	EncodeFunc     func(value interface{}) ([]byte, error)
	DecodeFunc     func(data []byte, value interface{}) error
	CompressFunc   func(data []byte) ([]byte, error)
	DecompressFunc func(compressed []byte) ([]byte, error)
	MarshalFunc    func(value interface{}) ([]byte, error)
	UnmarshalFunc  func(compressed []byte, value interface{}) error
}

func (c *Codec) Encode(value interface{}) ([]byte, error) {
	return c.EncodeFunc(value)
}

func (c *Codec) Decode(data []byte, value interface{}) error {
	return c.DecodeFunc(data, value)
}

func (c *Codec) Compress(data []byte) ([]byte, error) {
	return c.CompressFunc(data)
}

func (c *Codec) Decompress(data []byte) ([]byte, error) {
	return c.DecompressFunc(data)
}

func BaselineCodec(t *testing.T) *Codec {
	t.Helper()

	c := Codec{
		UnmarshalFunc: func(b []byte, v interface{}) error {
			return nil
		},
		MarshalFunc: func(v interface{}) ([]byte, error) {
			return []byte(`test`), nil
		},
	}

	return &c
}

func (c *Codec) Unmarshal(b []byte, v interface{}) error {
	return c.UnmarshalFunc(b, v)
}

func (c *Codec) Marshal(v interface{}) ([]byte, error) {
	return c.MarshalFunc(v)
}