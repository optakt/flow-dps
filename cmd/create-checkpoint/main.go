package main

import (
	"os"
	"time"

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

func run() int {

	// Parse the command line arguments.
	var (
		flagData       string
		flagIndex      string
		flagLevel      string
		flagCheckpoint string
	)

	pflag.StringVarP(&flagData, "data", "d", "", "database directory for protocol state")
	pflag.StringVarP(&flagIndex, "index", "i", "", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "directory for exporting the checkpoint")

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
		Str("data", flagData).
		Str("index", flagIndex).
		Str("level", flagLevel).
		Str("checkpoint_dir", flagCheckpoint).
		Msgf("flags loaded")

	// We should have at least one of data or index directories.
	if flagData == "" && flagIndex == "" {
		log.Error().Msg("need at least one of data or index directories")
		return failure
	}

	err = createCheckpoint(flagIndex, flagCheckpoint, log)
	if err != nil {
		log.Error().Msg("can not create checkpoint")
		return failure
	}

	return success
}
