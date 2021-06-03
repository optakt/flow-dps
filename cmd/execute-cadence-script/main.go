package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk/client"
)

func main() {

	var (
		flagAPI    string
		flagScript string
		flagLevel  string
		flagHeight int64
	)

	pflag.StringVarP(&flagScript, "script", "s", "", "cadence script to execute")
	pflag.StringVarP(&flagAPI, "api", "a", "127.0.0.1:3569", "access node API address")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log level for JSON logger")
	pflag.Int64VarP(&flagHeight, "height", "h", -1, "height on which to execute script, -1 for last indexed height")

	pflag.Parse()

	zerolog.TimestampFunc = func() time.Time { return time.Now() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err).Str("level", flagLevel).Msg("could not parse log level")
	}

	log = log.Level(level)

	cli, err := client.New(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Str("api", flagAPI).Err(err).Msg("could not connect to the access node")
	}

	script, err := ioutil.ReadFile(flagScript)
	if err != nil {
		log.Fatal().Err(err).Str("script", flagScript).Msg("could not read script file")
	}

	var value cadence.Value
	if flagHeight == -1 {
		value, err = cli.ExecuteScriptAtLatestBlock(context.Background(), script, []cadence.Value{})
	} else {
		value, err = cli.ExecuteScriptAtBlockHeight(context.Background(), uint64(flagHeight), script, []cadence.Value{})
	}

	if err != nil {
		log.Fatal().Err(err).Msg("cadence script execution failed")
	}

	fmt.Printf("%s\n", value.String())

	os.Exit(0)
}
