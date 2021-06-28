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

package zbor

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
)

type Codec struct {
	encoder      cbor.EncMode
	compressor   *zstd.Encoder
	decompressor *zstd.Decoder
}

// NewCodec creates a new Codec.
func NewCodec() (*Codec, error) {

	options := cbor.CanonicalEncOptions()
	options.Time = cbor.TimeRFC3339Nano
	encoder, err := options.EncMode()
	if err != nil {
		return nil, fmt.Errorf("could not initialize encoder: %w", err)
	}

	compressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderDict(Dictionary),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize compressor: %w", err)
	}

	decompressor, err := zstd.NewReader(nil,
		zstd.WithDecoderDicts(Dictionary),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize decompressor: %w", err)
	}

	c := Codec{
		encoder:      encoder,
		compressor:   compressor,
		decompressor: decompressor,
	}

	return &c, nil
}

func (c *Codec) Unmarshal(compressed []byte, value interface{}) error {
	data, err := c.decompressor.DecodeAll(compressed, nil)
	if err != nil {
		return fmt.Errorf("could not decompress value: %w", err)
	}
	err = cbor.Unmarshal(data, value)
	if err != nil {
		return fmt.Errorf("could not decode value: %w", err)
	}
	return nil
}

func (c *Codec) Marshal(value interface{}) ([]byte, error) {
	data, err := c.encoder.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("could not encode value: %w", err)
	}
	compressed := c.compressor.EncodeAll(data, nil)
	return compressed, nil
}
