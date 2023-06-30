package util

import (
	"fmt"

	"github.com/onflow/flow-archive/models/archive"
)

func ValidateHeightIndexed(reader archive.Reader, height uint64) error {
	h, err := reader.Last()
	if err != nil {
		return fmt.Errorf("could not get last indexed height for Archive node")
	}
	if height > h {
		return fmt.Errorf("the requested height (%d) is beyond the highest indexed height(%d)", height, h)
	}
	return nil
}

func ValidateRegisterHeightIndexed(reader archive.Reader, height uint64) error {
	h, err := reader.LatestRegisterHeight()
	if err != nil {
		return fmt.Errorf("could not get last indexed register height for Archive node")
	}
	if height > h {
		return fmt.Errorf("the requested height (%d) is beyond the highest indexed height(%d) for registers", height, h)
	}
	return nil
}
