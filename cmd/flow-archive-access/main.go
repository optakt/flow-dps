package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	grpczerolog "github.com/grpc-ecosystem/go-grpc-middleware/providers/zerolog/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"

	"github.com/onflow/flow/protobuf/go/flow/access"

	archiveAPI "github.com/onflow/flow-archive/api/archive"
	"github.com/onflow/flow-archive/codec/zbor"
	accessApi "github.com/onflow/flow-archive/service/access"
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
	signal.Notify(sig, syscall.SIGINT)

	// Command line parameter initialization.
	var (
		flagAddress string
		flagArchive string
		flagCache   uint64
		flagLevel   string
	)

	pflag.StringVarP(&flagAddress, "address", "a", "127.0.0.1:9000", "address to serve Access API on")
	pflag.StringVarP(&flagArchive, "archive", "d", "127.0.0.1:80", "host URL for Archive API endpoint")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")

	pflag.Uint64Var(&flagCache, "cache-size", 1_000_000_000, "maximum cache size for register reads in bytes")

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

	// Initialize codec.
	codec := zbor.NewCodec()

	// GRPC API initialization.
	opts := []logging.Option{
		logging.WithLevels(logging.DefaultServerCodeToLevel),
	}
	gsvr := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tags.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(grpczerolog.InterceptorLogger(log), opts...),
		),
		grpc.ChainStreamInterceptor(
			tags.StreamServerInterceptor(),
			logging.StreamServerInterceptor(grpczerolog.InterceptorLogger(log), opts...),
		),
	)

	// automatically add metrics with grpc_server_handled_total{grpc_code="Internal|Unknown|OK"}
	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(gsvr)

	// Initialize the API client.
	conn, err := grpc.Dial(flagArchive, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error().Str("dps", flagArchive).Err(err).Msg("could not dial API host")
		return failure
	}
	defer conn.Close()

	client := archiveAPI.NewAPIClient(conn)
	index := archiveAPI.IndexFromAPI(client, codec)

	invoke, err := invoker.New(index, invoker.WithCacheSize(flagCache))
	if err != nil {
		log.Error().Err(err).Msg("could not initialize script invoker")
		return failure
	}

	server := accessApi.NewServer(index, codec, invoke)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	listener, err := net.Listen("tcp", flagAddress)
	if err != nil {
		log.Error().Str("address", flagAddress).Err(err).Msg("could not listen")
		return failure
	}
	done := make(chan struct{})
	failed := make(chan struct{})
	go func() {
		log.Info().Msg("Flow Access API Server starting")

		access.RegisterAccessAPIServer(gsvr, server)
		err = gsvr.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn().Err(err).Msg("Flow Access API Server failed")
			close(failed)
		} else {
			close(done)
		}
		log.Info().Msg("Flow Access API Server stopped")
	}()

	select {
	case <-sig:
		log.Info().Msg("Flow Access API Server stopping")
	case <-done:
		log.Info().Msg("Flow Access API Server done")
	case <-failed:
		log.Warn().Msg("Flow Access API Server aborted")
		return failure
	}
	go func() {
		<-sig
		log.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	// The following code starts a shut down with a certain timeout and makes
	// sure that the main executing components are shutting down within the
	// allocated shutdown time. Otherwise, we will force the shutdown and log
	// an error. We then wait for shutdown on each component to complete.
	gsvr.GracefulStop()

	return success
}
