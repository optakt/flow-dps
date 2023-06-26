package main

import (
	"fmt"
	"os"
	"time"

	"github.com/onflow/flow-archive/service/storage2"
	"github.com/onflow/flow-go/model/flow"
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
		flagHeight uint64
		flagIndex  string
		flagLevel  string
		flagOwner  string
		flagKey    string
	)

	pflag.StringVarP(&flagIndex, "index", "i", "", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagOwner, "owner", "o", "", "owner in hex format")
	pflag.StringVarP(&flagKey, "key", "k", "", "register key in hex format")
	pflag.Uint64VarP(&flagHeight, "height", "i", 0, "height for getting register id")

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
		Uint64("height", flagHeight).
		Str("owner", flagOwner).
		Str("key", flagKey).
		Msgf("flags loaded")

	regID := flow.NewRegisterID(flagOwner, flagKey)

	err = GetPayload(flagIndex, flagHeight, regID, log)
	if err != nil {
		log.Error().Err(err).Msg("can not create checkpoint")
		return failure
	}

	return success
}

func GetPayload(indexDir string, height uint64, regID flow.RegisterID, log zerolog.Logger) error {
	lib2, err := storage2.NewLibrary2(indexDir, 1<<30)
	if err != nil {
		return err
	}

	regValue, err := lib2.GetPayload(height, regID)
	if err != nil {
		return fmt.Errorf("could not get register value: %w", err)
	}

	log.Info().Msgf("successfully get register value at height %v for reg id: %v: %x", height, regID, regValue)
	return nil
}
