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

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/utils/io"
	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/service/storage"
)

func generatePayloadSamples(db *badger.DB, size int) error {
	err := os.MkdirAll(updatesSamplePath, 0777)
	if err != nil {
		return fmt.Errorf("could not create update sample path: %w", err)
	}

	payloads, err := getPayloadSamples(db, size)
	if err != nil {
		return fmt.Errorf("could not retrieve payload sample: %w", err)
	}

	fmt.Printf(">>> Generated %d samples\n", len(payloads))

	for i, payload := range payloads {
		filename := filepath.Join(updatesSamplePath, fmt.Sprint(i))
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("could not create sample: %w", err)
		}

		_, err = file.Write(payload)
		if err != nil {
			_ = file.Close()
			return fmt.Errorf("could not write sample: %w", err)
		}

		_ = file.Close()
	}

	return nil
}

func getPayloadSamples(db *badger.DB, size int) ([][]byte, error) {
	codec := zbor.NewCodec()

	payloads := make([][]byte, 0, size)
	prefix := storage.EncodeKey(storage.PrefixPayload)
	err := db.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.IteratorOptions{
			Prefix:         prefix,
		})
		defer it.Close()

		it.Seek(prefix)

		var totalBytes int
		for totalBytes <= size {
			key := it.Item().Key()

			// If we're out of the payload entries, reset the iterator.
			if key[0] != storage.PrefixPayload {
				fmt.Println(">>> rewind")
				it.Rewind()
				it.Next()
			}

			val, err := tx.Get(key)
			if err != nil {
				return err
			}

			err = val.Value(func(val []byte) error {
				value, err := codec.Decompress(val)
				if err != nil {
					return err
				}

				if len(value) == 0 {
					return nil
				}

				payloads = append(payloads, value)
				totalBytes += len(value)

				return nil
			})
			if err != nil {
				return err
			}

			it.Next()
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return payloads, nil
}

func trainPayloadDictionary(size int) error {
	path := filepath.Join(updatesSamplePath, "*")
	samples, err := filepath.Glob(path)
	if err != nil {
		return fmt.Errorf("could not find any samples in path %s: %w", path, err)
	}

	command := []string{"zstd", "--train", "--maxdict", fmt.Sprint(size), "-o", updatesDictionaryPath}
	fmt.Printf(">>> Running command: %v <samples>\n", command)
	command = append(command, samples...)

	train := exec.Command(command[0], command[1:]...)

	output, err := train.CombinedOutput()
	fmt.Printf(">>> Got output: %s\n", output)
	if err != nil {
		return err
	}

	return nil
}

func benchmarkPayloadDictionary(db *badger.DB) (float64, time.Duration, error) {
	dict, err := io.ReadFile(updatesDictionaryPath)
	if err != nil {
		return 0, 0, fmt.Errorf("could not read dictionary: %w", err)
	}

	fmt.Println("dict size", len(dict))

	compressor, err := zstd.NewWriter(nil, zstd.WithEncoderDict(dict))
	if err != nil {
		return 0, 0, fmt.Errorf("could not create zstd writer: %w", err)
	}

	// Get 100kb worth of samples.
	payloads, err := getPayloadSamples(db, 100 * 1024)
	if err != nil {
		return 0, 0, fmt.Errorf("could not retrieve payloads: %w", err)
	}

	start := time.Now()
	var uncompressed int
	var compressed int
	for i := 0; i < 10000; i++ {
		// Pick a random payload.
		payload := payloads[rand.Int()%len(payloads)]

		uncompressed += len(payload)
		compressed += len(compressor.EncodeAll(payload, nil))
	}

	fmt.Println("compressed", compressed)
	fmt.Println("uncompressed", uncompressed)
	ratio := float64(compressed) / float64(uncompressed)

	return ratio, time.Since(start), nil
}
