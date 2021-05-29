package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

func main() {

	var (
		flagDir      string
		flagLogLevel string
	)

	pflag.StringVarP(&flagDir, "dir", "d", "", "path to badger database")
	pflag.StringVarP(&flagLogLevel, "log-level", "l", "info", "log level for JSON logger")

	pflag.Parse()

	zerolog.TimestampFunc = func() time.Time { return time.Now() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLogLevel)
	if err != nil {
		log.Fatal().Err(err)
	}

	log = log.Level(level)

	if flagDir == "" {
		log.Fatal().Msg("path to badger database is required")
	}

	opts := badger.DefaultOptions(flagDir).WithReadOnly(true)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open badger db")
	}

	defer db.Close()

	var buf bytes.Buffer

	compressor, err := zstd.NewWriter(&buf)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize zstd compression")
	}

	_, err = db.Backup(compressor, 0)
	if err != nil {
		// clean up the encoder resources; OS would do it anyway but no harm in being explicit
		compressor.Close()

		log.Fatal().Err(err).Msg("could not backup badger db")
	}

	err = compressor.Close()
	if err != nil {
		log.Warn().Err(err).Msg("could not close compression writer")
	}

	fmt.Printf("%s", hex.EncodeToString(buf.Bytes()))
}
