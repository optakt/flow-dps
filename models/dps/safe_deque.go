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

package dps

import (
	"sync"

	"github.com/gammazero/deque"
)

type SafeDeque struct {
	mutex *sync.Mutex
	deque *deque.Deque
}

func NewDeque() *SafeDeque {
	s := SafeDeque{
		mutex: &sync.Mutex{},
		deque: deque.New(),
	}
	return &s
}

func (s *SafeDeque) Len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Len()
}

func (s *SafeDeque) Cap() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Cap()
}

func (s *SafeDeque) Front() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Front()
}

func (s *SafeDeque) Back() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Back()
}

func (s *SafeDeque) PushFront(v interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.PushFront(v)
}

func (s *SafeDeque) PushBack(v interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.PushBack(v)
}

func (s *SafeDeque) PopFront() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.PopFront()
}

func (s *SafeDeque) PopBack() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.PopBack()
}

func (s *SafeDeque) Rotate(n int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.Rotate(n)
}

func (s *SafeDeque) Set(i int, v interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.Set(i, v)
}

func (s *SafeDeque) SetMinCapacity(cap uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.SetMinCapacity(cap)
}

func (s *SafeDeque) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.Clear()
}
