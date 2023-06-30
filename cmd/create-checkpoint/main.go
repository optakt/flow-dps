package main

import (
	"os"
	"time"

	"github.com/onflow/flow-archive/service/storage2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

const (
	success = 0
	failure = 1
)

func main() {
	os.Exit(run())
}

// create-checkpoint command creates a checkpoint for the payload database
func run() int {

	// Parse the command line arguments.
	var (
		flagIndex      string
		flagLevel      string
		flagCheckpoint string
	)

	pflag.StringVarP(&flagIndex, "index", "i", "/var/flow/data/pebble/index2", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "directory for exporting the checkpoint, use a different folder than /var/flow/data/pebble")

	pflag.Parse()

	// Initialize the logger.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	log.Info().
		Str("index", flagIndex).
		Str("level", flagLevel).
		Str("checkpoint_dir", flagCheckpoint).
		Msgf("flags loaded")

	// We should have at least one of data or index directories.
	if flagIndex == "" {
		log.Error().Msg("missing index directory")
		return failure
	}

	storagePath := storage2.StoragePath(flagIndex)
	// Check if the path exists
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		log.Error().Msgf("The storagePath '%s' does not exist.\n", storagePath)
		return failure
	}

	if flagCheckpoint == "" {
		log.Error().Msg("missing checkpoint directory")
		return failure
	}

	if flagCheckpoint == flagIndex {
		log.Error().Msgf("checkpoint must be a different directory than the index folder")
		return failure
	}

	err = createCheckpoint(flagIndex, flagCheckpoint, log)
	if err != nil {
		log.Error().Err(err).Msg("can not create checkpoint")
		return failure
	}

	log.Info().Msgf("successfully created checkpoint at dir: %v", flagCheckpoint)

	return success
}
