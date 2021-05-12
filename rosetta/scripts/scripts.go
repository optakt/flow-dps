package scripts

import (
	"github.com/onflow/flow-go/model/flow"
)

func WithParams(params Params) func(*Scripts) {
	return func(s *Scripts) {
		s.params = params
	}
}

func WithToken(symbol string, address flow.Address) func(*Scripts) {
	return func(s *Scripts) {
		s.tokens[symbol] = address
	}
}

type Scripts struct {
	params Params
	tokens map[string]flow.Address
}

func New(options ...func(*Scripts)) *Scripts {

	s := Scripts{
		params: TestNet(),
		tokens: make(map[string]flow.Address),
	}

	for _, option := range options {
		option(&s)
	}

	return &s
}
