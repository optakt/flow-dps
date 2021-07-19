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

type Cache struct {
	GetFunc func(key interface{}) (interface{}, bool)
	SetFunc func(key, value interface{}, cost int64) bool
}

func BaselineCache(t *testing.T) *Cache {
	t.Helper()

	c := Cache{
		GetFunc: func(interface{}) (interface{}, bool) {
			return GenericBytes, true
		},
		SetFunc: func(interface{}, interface{}, int64) bool {
			return true
		},
	}

	return &c
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	return c.GetFunc(key)
}

func (c *Cache) Set(key, value interface{}, cost int64) bool {
	return c.SetFunc(key, value, cost)
}
