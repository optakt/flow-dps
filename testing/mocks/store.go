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

type Store struct {
	SaveFunc     func(key [32]byte, payload []byte) error
	CachedFunc   func(key [32]byte) bool
	RetrieveFunc func(key [32]byte) ([]byte, error)
	CloseFunc    func() error
}

func BaselineStore() *Store {
	s := Store{
		SaveFunc: func(key [32]byte, payload []byte) error { return nil },
		RetrieveFunc: func(key [32]byte) ([]byte, error) {
			return GenericLedgerPayload(0).Value[:], nil
		},
		CloseFunc: func() error { return nil },
	}

	return &s
}

func (s *Store) Save(key [32]byte, payload []byte) error {
	return s.SaveFunc(key, payload)
}

func (s *Store) Cached(key [32]byte) bool {
	return s.CachedFunc(key)
}

func (s *Store) Retrieve(key [32]byte) ([]byte, error) {
	return s.RetrieveFunc(key)
}

func (s *Store) Close() error {
	return s.CloseFunc()
}
