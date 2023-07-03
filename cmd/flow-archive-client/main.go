package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/onflow/cadence/encoding/json"

	"github.com/onflow/flow-archive/api/archive"
	"github.com/onflow/flow-archive/codec/zbor"
	"github.com/onflow/flow-archive/models/convert"
	"github.com/onflow/flow-archive/service/invoker"
)

const (
	success = 0
	failure = 1
)

func main() {
	os.Exit(run())
}

func run() int {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagAPI    string
		flagCache  uint64
		flagHeight uint64
		flagLevel  string
		flagParams string
		flagScript string
	)

	pflag.StringVarP(&flagAPI, "api", "a", "", "host for GRPC API server")
	pflag.Uint64VarP(&flagCache, "cache", "e", 1_000_000_000, "maximum cache size for register reads in bytes")
	pflag.Uint64VarP(&flagHeight, "height", "h", 0, "block height to execute the script at")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagParams, "params", "p", "", "comma-separated list of Cadence parameters")
	pflag.StringVarP(&flagScript, "script", "s", "script.cdc", "path to file with Cadence script")

	pflag.Parse()

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// If no API server is given, choose based on height.
	if flagAPI == "" {
		for _, spork := range DefaultSporks {
			if flagHeight >= spork.First && flagHeight <= spork.Last {
				log.Info().Uint64("height", flagHeight).Str("spork", spork.Name).Str("api", spork.API).Msg("spork and API chosen based on height")
				flagAPI = spork.API
				break
			}
		}
	}
	if flagAPI == "" {
		log.Error().Uint64("height", flagHeight).Msg("could not find spork and API for height")
		return failure
	}

	// Initialize the API client.
	conn, err := grpc.Dial(flagAPI)
	if err != nil {
		log.Error().Str("api", flagAPI).Err(err).Msg("could not dial API host")
		return failure
	}
	defer conn.Close()

	// Read the script.
	script, err := os.ReadFile(flagScript)
	if err != nil {
		log.Error().Str("script", flagScript).Err(err).Msg("could not read script")
		return failure
	}

	// Decode the arguments
	var args [][]byte
	if flagParams != "" {
		params := strings.Split(flagParams, ",")
		for _, param := range params {
			carg, err := convert.ParseCadenceArgument(param)
			if err != nil {
				log.Error().Err(err).Msg("invalid Cadence value")
				return failure
			}
			arg, err := json.Encode(carg)
			args = append(args, arg)
		}
	}

	// Initialize codec.
	codec := zbor.NewCodec()

	// Execute the script using remote lookup and read.
	client := archive.NewAPIClient(conn)
	invoke, err := invoker.New(archive.IndexFromAPI(client, codec), invoker.WithCacheSize(flagCache))
	if err != nil {
		log.Error().Err(err).Msg("could not initialize invoker")
		return failure
	}
	result, err := invoke.Script(flagHeight, script, args)
	if err != nil {
		log.Error().Err(err).Msg("could not invoke script")
		return failure
	}
	output, err := json.Encode(result)
	if err != nil {
		log.Error().Uint64("height", flagHeight).Err(err).Msg("could not encode result")
		return failure
	}

	fmt.Println(string(output))

	return success
}
