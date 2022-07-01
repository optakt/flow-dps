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

// SafeDeque is a concurrency-safe double-ended queue.
// NOTE: As specified in the original Deque documentation, concurrency
// safety is up to the consumer to provide.
// See https://github.com/gammazero/deque
type SafeDeque struct {
	mutex *sync.Mutex
	deque *deque.Deque
}

// NewDeque instantiates and returns a new empty double-ended queue.
func NewDeque() *SafeDeque {
	s := SafeDeque{
		mutex: &sync.Mutex{},
		deque: deque.New(),
	}
	return &s
}

// Len returns the length of the queue.
func (s *SafeDeque) Len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Len()
}

// Cap returns the capacity of the queue.
func (s *SafeDeque) Cap() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Cap()
}

// Front returns the element at the front of the queue.
// It panics if the queue is empty.
func (s *SafeDeque) Front() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Front()
}

// Back returns the element at the back of the queue.
// It panics if the queue is empty.
func (s *SafeDeque) Back() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.Back()
}

// PushFront prepends an element to the front of the queue.
func (s *SafeDeque) PushFront(v interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.PushFront(v)
}

// PushBack appends an element to the back of the  queue.
func (s *SafeDeque) PushBack(v interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.PushBack(v)
}

// PopFront removes and returns the element from the front of the queue.
func (s *SafeDeque) PopFront() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.PopFront()
}

// PopBack removes and returns the element from the back of the queue.
func (s *SafeDeque) PopBack() interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.deque.PopBack()
}

// Rotate rotates the deque n steps front-to-back.
func (s *SafeDeque) Rotate(n int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.Rotate(n)
}

// Set puts the element at index i in the queue.
func (s *SafeDeque) Set(i int, v interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.Set(i, v)
}

// SetMinCapacity sets a minimum capacity of 2^cap.
// If the value of the minimum capacity is less than
// or equal to the minimum allowed, then capacity is
// set to the minimum allowed.
func (s *SafeDeque) SetMinCapacity(cap uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.SetMinCapacity(cap)
}

// Clear removes all elements from the queue, but retains the current capacity.
func (s *SafeDeque) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deque.Clear()
}
