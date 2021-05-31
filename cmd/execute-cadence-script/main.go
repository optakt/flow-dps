package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk/client"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

func main() {

	var (
		flagAPI        string
		flagScriptFile string
		flagLogLevel   string
		flagHeight     int64
	)

	// TODO: mismatch between block height type - using int64 instead of uint64

	pflag.StringVarP(&flagScriptFile, "script-file", "s", "", "cadence script to execute")
	pflag.StringVarP(&flagAPI, "api", "a", "localhost:3569", "access node API address")
	pflag.StringVarP(&flagLogLevel, "log-level", "l", "info", "log level for JSON logger")
	pflag.Int64VarP(&flagHeight, "height", "h", -1, "height on which to execute script")

	pflag.Parse()

	zerolog.TimestampFunc = func() time.Time { return time.Now() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLogLevel)
	if err != nil {
		log.Fatal().Err(err)
	}

	log = log.Level(level)

	cli, err := client.New(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to the access node")
	}

	script, err := ioutil.ReadFile(flagScriptFile)
	if err != nil {
		log.Fatal().Err(err).Str("file", flagScriptFile).Msg("could not read script file")
	}

	var value cadence.Value
	if flagHeight == -1 {
		value, err = cli.ExecuteScriptAtLatestBlock(context.Background(), script, []cadence.Value{})
	} else {
		value, err = cli.ExecuteScriptAtBlockHeight(context.Background(), uint64(flagHeight), script, []cadence.Value{})
	}

	if err != nil {
		log.Error().Err(err).Msg("cadence script execution failed")
		return
	}

	fmt.Printf("%s\n", value.String())

}
