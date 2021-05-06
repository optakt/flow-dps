package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"

	"github.com/awfm9/flow-dps/chain"
	"github.com/awfm9/flow-dps/feeder"
	"github.com/awfm9/flow-dps/indexer"
	"github.com/awfm9/flow-dps/ledger"
	"github.com/awfm9/flow-dps/mapper"
	"github.com/awfm9/flow-dps/rest"
)

func main() {

	var (
		flagLevel      string
		flagData       string
		flagTrie       string
		flagIndex      string
		flagCheckpoint string
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", "log output level")
	pflag.StringVarP(&flagData, "data-dir", "d", "data", "protocol state database directory")
	pflag.StringVarP(&flagTrie, "trie-dir", "t", "trie", "state trie write-ahead log directory")
	pflag.StringVarP(&flagIndex, "index-dir", "i", "index", "state ledger index directory")
	pflag.StringVarP(&flagCheckpoint, "checkpoint-file", "c", "", "state trie root checkpoint file")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	chain, err := chain.FromProtocolState(flagData)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize chain")
	}

	feeder, err := feeder.FromLedgerWAL(flagTrie)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize feeder")
	}

	indexer, err := indexer.New(flagIndex)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize indexer")
	}

	mapper, err := mapper.New(log, chain, feeder, indexer, mapper.WithCheckpointFile(flagCheckpoint))
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize mapper")
	}

	core, err := ledger.NewCore(indexer.DB())
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize ledger")
	}

	controller, err := rest.NewController(core)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize controller")
	}

	e := echo.New()
	e.GET("/registers/:key", controller.GetRegister)
	e.GET("/payloads/:keys", controller.GetPayloads)

	err = mapper.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("could not run mapper")
	}
}
