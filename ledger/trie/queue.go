package trie

import (
	"github.com/gammazero/deque"
)

type queue struct {
	nodes *deque.Deque
}

func newQueue() *queue {
	return &queue{
		nodes: deque.New(256),
	}
}

func (q *queue) Push(n Node) {
	q.nodes.PushFront(n)
}

func (q *queue) Pop() Node {
	n := q.nodes.PopFront().(Node)
	return n
}

func (q *queue) Len() int {
	return q.nodes.Len()
}