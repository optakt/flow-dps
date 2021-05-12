package scripts

import (
	"strings"

	"github.com/onflow/flow-go/model/flow"
)

func WithParams(params Params) func(*Scripts) {
	return func(s *Scripts) {
		s.params = params
	}
}

type Scripts struct {
	params Params
}

func New(options ...func(*Scripts)) *Scripts {

	s := Scripts{
		params: TestNet(),
	}

	for _, option := range options {
		option(&s)
	}

	return &s
}

func (s *Scripts) GetBalance(token flow.Address) []byte {
	script := GetBalance
	script = strings.ReplaceAll(script, PlaceholderFungible, s.params.FungibleToken.Hex())
	script = strings.ReplaceAll(script, PlaceholderToken, token.Hex())
	return []byte(script)
}
