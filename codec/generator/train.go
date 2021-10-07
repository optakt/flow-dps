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

package generator

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func (g *Generator) trainDictionary(kind DictionaryKind, size int) (*dictionary, error) {

	// List all samples within the sample path, to be given to the training command.
	path := filepath.Join(g.cfg.samplePath, string(kind), "*")
	samples, err := filepath.Glob(path)
	if err != nil {
		return nil, fmt.Errorf("could not find any samples in path %s: %w", path, err)
	}

	// Build the training command.
	rawDictPath := filepath.Join(g.cfg.dictionaryPath, string(kind))
	command := []string{"zstd", "--train", "--maxdict", fmt.Sprint(size), "-o", rawDictPath}
	command = append(command, samples...)

	train := exec.Command(command[0], command[1:]...)

	// Run the training.
	err = train.Run()
	if err != nil {
		return nil, fmt.Errorf("could not train dictionary: %w", err)
	}

	// Open and read the resulting raw dictionary.
	rawDict, err := os.Open(rawDictPath)
	if err != nil {
		return nil, fmt.Errorf("could not open raw dictionary: %w", err)
	}

	raw, err := io.ReadAll(rawDict)
	if err != nil {
		return nil, fmt.Errorf("could not read raw dictionary: %w", err)
	}

	// Remove raw dictionary since we have its bytes in memory.
	err = os.RemoveAll(rawDictPath)
	if err != nil {
		return nil, fmt.Errorf("could not delete raw dictionary from filesystem: %w", err)
	}

	dict := dictionary{
		kind: kind,
		raw:  raw,
		size: size,
	}

	return &dict, nil
}
