package trie

import (
	"sync"
)

type Pool struct {
	extensions *sync.Pool
	branches   *sync.Pool
	leaves     *sync.Pool
}

func NewPool(number int) *Pool {
	ePool := &sync.Pool{}
	ePool.New = func() interface{} {
		return &Extension{}
	}
	bPool := &sync.Pool{}
	bPool.New = func() interface{} {
		return &Branch{}
	}
	lPool := &sync.Pool{}
	lPool.New = func() interface{} {
		return &Leaf{}
	}

	// Pre allocate each node type.
	for i := 0; i < number; i++ {
		ePool.Put(ePool.New())
		bPool.Put(bPool.New())
		lPool.Put(lPool.New())
	}

	p := Pool{
		extensions: ePool,
		branches:   bPool,
		leaves:     lPool,
	}

	return &p
}
