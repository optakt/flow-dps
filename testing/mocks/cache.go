package mocks

import (
	"testing"
)

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
