package scripts

import (
	"github.com/onflow/flow-go/model/flow"
)

type Params struct {
	FungibleToken flow.Address
}

func TestNet() Params {
	return Params{
		FungibleToken: flow.HexToAddress("0x9a0766d93b6608b7"),
	}
}

func MainNet() Params {
	return Params{
		FungibleToken: flow.HexToAddress("0xf233dcee88fe0abe"),
	}
}
