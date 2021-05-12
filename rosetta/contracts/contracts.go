package contracts

import (
	"github.com/onflow/flow-go/model/flow"
)

func WithToken(symbol string, address flow.Address) func(*Contracts) {
	return func(c *Contracts) {
		c.tokens[symbol] = address
	}
}

type Contracts struct {
	tokens map[string]flow.Address
}

func New(options ...func(*Contracts)) *Contracts {

	c := Contracts{
		tokens: make(map[string]flow.Address),
	}

	for _, option := range options {
		option(&c)
	}

	return &c
}

func (c *Contracts) Token(symbol string) (flow.Address, bool) {
	address, ok := c.tokens[symbol]
	return address, ok
}
