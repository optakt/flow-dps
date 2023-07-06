package invoker

import (
	"fmt"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-go/fvm/environment"
	"github.com/onflow/flow-go/fvm/errors"
	"github.com/onflow/flow-go/model/flow"
)

var _ environment.Blocks = (*Blocks)(nil)

type Blocks struct {
	index archive.Reader
}

// ByHeightFrom implements the fvm/env/blocks interface
func (b *Blocks) ByHeightFrom(height uint64, header *flow.Header) (*flow.Header, error) {

	if header.Height == height {
		return header, nil
	}

	if height > header.Height {
		minHeight, err := b.index.First()
		if err != nil {
			return nil, fmt.Errorf("could not find first indexed height: %w", err)
		}
		// imitate flow-go error format
		err = errors.NewValueErrorf(fmt.Sprint(height),
			"requested height (%d) is not in the range(%d, %d)", height, minHeight, header.Height)
		return nil, fmt.Errorf("cannot retrieve block parent: %w", err)
	}
	// find the block in storage as all of them are guaranteed to be finalized
	return b.index.Header(height)
}

func NewBlocks(index archive.Reader) *Blocks {
	return &Blocks{
		index: index,
	}
}
