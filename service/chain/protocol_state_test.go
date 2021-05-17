package chain_test

import (
	"math"
	"os"
	"testing"

	"github.com/onflow/flow-go/model/flow"
	"github.com/stretchr/testify/assert"

	"github.com/awfm9/flow-dps/service/chain"
)

var c *chain.ProtocolState

func TestMain(m *testing.M) {
	var err error
	c, err = chain.FromProtocolState("./testdata")
	if err != nil {
		panic(err)
	}
	code := m.Run()
	os.Exit(code)
}

func TestFromProtocolState(t *testing.T) {
	assert.NotNil(t, c)
}

func TestProtocolState_Root(t *testing.T) {
	root, err := c.Root()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), root)
}

func TestProtocolState_Header(t *testing.T) {
	header, err := c.Header(0)
	assert.NoError(t, err)
	assert.Equal(t, flow.ChainID("flow-testnet"), header.ChainID)

	header, err = c.Header(math.MaxUint64)
	assert.Error(t, err)
}

func TestProtocolState_Commit(t *testing.T) {
	want := []byte{0xd8, 0x5b, 0x7d, 0xc2, 0xd6, 0xbe, 0x69, 0xc5, 0xcc, 0x10, 0xf0, 0xd1, 0x28, 0x59, 0x53, 0x52, 0x35, 0x4e, 0x57, 0xfb, 0xd9, 0x23, 0xac, 0x1a, 0xd3, 0xf7, 0x34, 0x51, 0x86, 0x10, 0xca, 0x73}

	commit, err := c.Commit(0)
	assert.NoError(t, err)
	assert.Equal(t, want, commit)

	commit, err = c.Commit(math.MaxUint64)
	assert.Error(t, err)
}

func TestProtocolState_Events(t *testing.T) {
	_, err := c.Events(0)
	assert.NoError(t, err)
	// TODO: Get a state with events to be able to test this.
	//assert.Len(t, events, 42)

	_, err = c.Events(math.MaxUint64)
	assert.Error(t, err)
}
