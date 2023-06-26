package main

import (
	"fmt"

	"github.com/onflow/flow-archive/service/storage2"
	"github.com/rs/zerolog"
)

func createCheckpoint(indexDir string, checkpointDir string, log zerolog.Logger) error {
	lib2, err := storage2.NewLibrary2(indexDir, 1<<30)
	if err != nil {
		return err
	}

	err = lib2.Checkpoint(checkpointDir)
	if err != nil {
		return fmt.Errorf("could not create checkpoint at dir (%v): %w", checkpointDir, err)
	}

	return nil
}
