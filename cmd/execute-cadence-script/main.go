package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk/client"
)

const (
	success = 0
	failure = -1
)

func main() {
	os.Exit(int(run()))
}

func run() int {

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
		log.Error().Err(err).Str("level", flagLevel).Msg("could not parse log level")
		return failure
	}

	log = log.Level(level)

	cli, err := client.New(flagAPI, grpc.WithInsecure())
	if err != nil {
		log.Error().Str("api", flagAPI).Err(err).Msg("could not connect to the access node")
		return failure
	}

	script, err := os.ReadFile(flagScript)
	if err != nil {
		log.Error().Err(err).Str("script", flagScript).Msg("could not read script file")
		return failure
	}

	var value cadence.Value
	if flagHeight == -1 {
		value, err = cli.ExecuteScriptAtLatestBlock(context.Background(), script, []cadence.Value{})
	} else {
		value, err = cli.ExecuteScriptAtBlockHeight(context.Background(), uint64(flagHeight), script, []cadence.Value{})
	}

	if err != nil {
		log.Error().Err(err).Msg("cadence script execution failed")
		return failure
	}

	fmt.Printf("%s\n", value.String())

	return success
}
