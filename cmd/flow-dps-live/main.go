// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package main

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	gcloud "cloud.google.com/go/storage"
	"github.com/dgraph-io/badger/v2"
	grpczerolog "github.com/grpc-ecosystem/go-grpc-middleware/providers/zerolog/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"google.golang.org/api/option"
	"google.golang.org/grpc"

	sdk "github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go/cmd/bootstrap/utils"
	"github.com/onflow/flow-go/crypto"
	unstaked "github.com/onflow/flow-go/follower"
	"github.com/onflow/flow-go/model/bootstrap"

	api "github.com/onflow/flow-dps/api/dps"
	"github.com/onflow/flow-dps/codec/zbor"
	"github.com/onflow/flow-dps/models/dps"
	"github.com/onflow/flow-dps/service/cloud"
	"github.com/onflow/flow-dps/service/forest"
	"github.com/onflow/flow-dps/service/index"
	"github.com/onflow/flow-dps/service/initializer"
	"github.com/onflow/flow-dps/service/loader"
	"github.com/onflow/flow-dps/service/mapper"
	"github.com/onflow/flow-dps/service/metrics"
	"github.com/onflow/flow-dps/service/storage"
	"github.com/onflow/flow-dps/service/tracker"
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
		flagAddress    string
		flagBootstrap  string
		flagBucket     string
		flagCheckpoint string
		flagData       string
		flagIndex      string
		flagLevel      string
		flagMetrics    string
		flagSkip       bool

		flagFlushInterval time.Duration
		flagSeedAddress   string
		flagSeedKey       string
	)

	pflag.StringVarP(&flagAddress, "address", "a", "127.0.0.1:5005", "bind address for serving DPS API")
	pflag.StringVarP(&flagBootstrap, "bootstrap", "b", "bootstrap", "path to directory with bootstrap information for spork")
	pflag.StringVarP(&flagBucket, "bucket", "u", "", "Google Cloude Storage bucket with block data records")
	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "", "path to root checkpoint file for execution state trie")
	pflag.StringVarP(&flagData, "data", "d", "data", "path to database directory for protocol data")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "path to database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.StringVarP(&flagMetrics, "metrics", "m", "", "address on which to expose metrics (no metrics are exposed when left empty)")
	pflag.BoolVarP(&flagSkip, "skip", "s", false, "skip indexing of execution state ledger registers")

	pflag.DurationVar(&flagFlushInterval, "flush-interval", 1*time.Second, "interval for flushing badger transactions (0s for disabled)")
	pflag.StringVar(&flagSeedAddress, "seed-address", "", "host address of seed node to follow consensus")
	pflag.StringVar(&flagSeedKey, "seed-key", "", "hex-encoded public network key of seed node to follow consensus")

	pflag.Parse()

	// Increase the GOMAXPROCS value in order to use the full IOPS available, see:
	// https://groups.google.com/g/golang-nuts/c/jPb_h3TvlKE
	_ = runtime.GOMAXPROCS(128)

	// Logger initialization.
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Error().Str("level", flagLevel).Err(err).Msg("could not parse log level")
		return failure
	}
	log = log.Level(level)

	// As a first step, we will open the protocol state and the index database.
	// The protocol state database is what the consensus follower will write to
	// and the mapper will read from. The index database is what the mapper will
	// write to and the DPS API will read from.
	indexDB, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index database")
		return failure
	}
	defer func() {
		err := indexDB.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index database")
		}
	}()
	protocolDB, err := badger.Open(dps.DefaultOptions(flagData))
	if err != nil {
		log.Error().Err(err).Msg("could not open protocol state database")
		return failure
	}
	defer func() {
		err := protocolDB.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close protocol state database")
		}
	}()

	// Next, we initialize the index reader and writer. They use a common codec
	// and storage library to interact with the underlying database. If there
	// already is an index database, we need the force flag to be set, as we do
	// not want to start overwriting data in the index silently. We also need
	// to flush the writer to make sure all data is written correctly when
	// shutting down.
	codec := zbor.NewCodec()
	storage := storage.New(codec)
	read := index.NewReader(indexDB, storage)
	first, err := read.First()
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		log.Error().Err(err).Msg("could not get first height from index reader")
		return failure
	}
	empty := errors.Is(err, badger.ErrKeyNotFound)
	if empty && flagCheckpoint == "" {
		log.Error().Msg("index database is empty, please provide root checkpoint (-c, --checkpoint) to bootstrap")
		return failure
	}

	// We initialize the writer with a flush interval, which will make sure that
	// Badger transactions are committed to the database, even if they don't
	// fill up fast enough. This avoids having latency between when we add data
	// to the transaction and when it becomes available on-disk for serving the
	// DPS API.
	write := index.NewWriter(
		indexDB,
		storage,
		index.WithFlushInterval(flagFlushInterval),
	)

	defer func() {
		err := write.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index writer")
		}
	}()

	// Next, we want to initialize the consensus follower. One needed parameter
	// is a network key, used to secure the peer-to-peer communication. However,
	// as we do not need any specific key, we choose to just initialize a new
	// key on each start of the live indexer.
	seed := make([]byte, crypto.KeyGenSeedMinLenECDSASecp256k1)
	n, err := rand.Read(seed)
	if err != nil || n != crypto.KeyGenSeedMinLenECDSASecp256k1 {
		log.Error().Err(err).Msg("could not generate private key seed")
		return failure
	}
	privKey, err := utils.GenerateUnstakedNetworkingKey(seed)
	if err != nil {
		log.Error().Err(err).Msg("could not generate private network key")
		return failure
	}

	// Here, we finally initialize the unstaked consensus follower. It connects
	// to a staked access node for bootstrapping the peer-to-peer network, which
	// is shared between staked access nodes and unstaked consensus followers.
	// For every finalized block, it calls the callback for all registered
	// finalization listeners.
	seedHost, port, err := net.SplitHostPort(flagSeedAddress)
	if err != nil {
		log.Error().Err(err).Str("address", flagSeedAddress).Msg("could not parse seed node address")
		return failure
	}
	seedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		log.Error().Err(err).Str("port", port).Msg("could not parse seed node port")
		return failure
	}
	seedKey, err := sdk.DecodePublicKeyHex(sdk.ECDSA_P256, flagSeedKey)
	if err != nil {
		log.Error().Err(err).Str("key", flagSeedKey).Msg("could not parse seed node network public key")
		return failure
	}
	seedNodes := []unstaked.BootstrapNodeInfo{{
		Host:             seedHost,
		Port:             uint(seedPort),
		NetworkPublicKey: seedKey,
	}}
	follow, err := unstaked.NewConsensusFollower(
		privKey,
		"0.0.0.0:0", // automatically choose port, listen on all IPs
		seedNodes,
		unstaked.WithBootstrapDir(flagBootstrap),
		unstaked.WithDB(protocolDB),
		unstaked.WithLogLevel(flagLevel),
	)
	if err != nil {
		log.Error().Err(err).Str("bucket", flagBucket).Msg("could not create consensus follower")
		return failure
	}

	// There is a problem with the Flow consensus follower API which makes it
	// impossible to use it to bootstrap the protocol state. The consensus
	// follower will only bootstrap it when it's starting. This makes it
	// impossible to initialize our consensus tracker, which needs a valid
	// protocol state, and to add it to the consensus follower for block
	// finalization, without missing some blocks. As a work-around, we manually
	// bootstrap the Flow protocol state using the bootstrap data here.
	path := filepath.Join(flagBootstrap, bootstrap.PathRootProtocolStateSnapshot)
	file, err := os.Open(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("could not open protocol state snapshot")
		return failure
	}
	defer file.Close()
	err = initializer.ProtocolState(file, protocolDB)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize protocol state")
		return failure
	}

	// If we are resuming, and the consensus follower has already finalized some
	// blocks that were not yet indexed, we need to download them again in the
	// cloud streamer. Here, we figure out which blocks these are.
	blockIDs, err := initializer.CatchupBlocks(protocolDB, read)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize catch-up blocks")
		return failure
	}

	// On the other side, we also need access to the execution data. The cloud
	// streamer is responsible for retrieving block execution records from a
	// Google Cloud Storage bucket. This component plays the role of what would
	// otherwise be a network protocol, such as a publish socket.
	client, err := gcloud.NewClient(context.Background(),
		option.WithoutAuthentication(),
	)
	if err != nil {
		log.Error().Err(err).Msg("could not connect GCP client")
		return failure
	}
	defer func() {
		err := client.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close GCP client")
		}
	}()
	bucket := client.Bucket(flagBucket)
	stream := cloud.NewGCPStreamer(log, bucket,
		cloud.WithCatchupBlocks(blockIDs),
	)

	// Next, we can initialize our consensus and execution trackers. They are
	// responsible for tracking changes to the available data, for the consensus
	// follower and related consensus data on one side, and the cloud streamer
	// and available execution records on the other side.
	execution, err := tracker.NewExecution(log, protocolDB, stream)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize execution tracker")
		return failure
	}
	consensus, err := tracker.NewConsensus(log, protocolDB, execution)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize consensus tracker")
		return failure
	}

	// We can now register the consensus tracker and the cloud streamer as
	// finalization listeners with the consensus follower. The consensus tracker
	// will use the callback to make additional data available to the mapper,
	// while the cloud streamer will use the callback to download execution data
	// for finalized blocks.
	follow.AddOnBlockFinalizedConsumer(stream.OnBlockFinalized)
	follow.AddOnBlockFinalizedConsumer(consensus.OnBlockFinalized)

	// If we have an empty database, we want a loader to bootstrap from the
	// checkpoint; if we don't, we can optionally use the root checkpoint to
	// speed up the restart/restoration.
	var load mapper.Loader
	load = loader.FromIndex(log, storage, indexDB)
	if empty {
		load = loader.FromCheckpointFile(flagCheckpoint, &log)
	} else if flagCheckpoint != "" {
		initialize := loader.FromCheckpointFile(flagCheckpoint, &log)
		load = loader.FromIndex(log, storage, indexDB,
			loader.WithInitializer(initialize),
			loader.WithExclude(loader.ExcludeAtOrBelow(first)),
		)
	}

	// If metrics are enabled, the mapper should use the metrics writer. Otherwise, it can
	// use the regular one.
	writer := dps.Writer(write)
	metricsEnabled := flagMetrics != ""
	if metricsEnabled {
		writer = index.NewMetricsWriter(write)
	}

	// At this point, we can initialize the core business logic of the indexer,
	// with the mapper's finite state machine and transitions. We also want to
	// load and inject the root checkpoint if it is given as a parameter.
	transitions := mapper.NewTransitions(log, load, consensus, execution, read, writer,
		mapper.WithBootstrapState(empty),
		mapper.WithSkipRegisters(flagSkip),
	)
	forest := forest.New()
	state := mapper.EmptyState(forest)
	fsm := mapper.NewFSM(state,
		mapper.WithTransition(mapper.StatusInitialize, transitions.InitializeMapper),
		mapper.WithTransition(mapper.StatusBootstrap, transitions.BootstrapState),
		mapper.WithTransition(mapper.StatusResume, transitions.ResumeIndexing),
		mapper.WithTransition(mapper.StatusIndex, transitions.IndexChain),
		mapper.WithTransition(mapper.StatusUpdate, transitions.UpdateTree),
		mapper.WithTransition(mapper.StatusCollect, transitions.CollectRegisters),
		mapper.WithTransition(mapper.StatusMap, transitions.MapRegisters),
		mapper.WithTransition(mapper.StatusForward, transitions.ForwardHeight),
	)

	// Next, we initialize the GRPC server that will serve the DPS API on top of
	// the index database that is generated live by the mapper.
	logOpts := []logging.Option{
		logging.WithLevels(logging.DefaultServerCodeToLevel),
	}
	interceptor := grpczerolog.InterceptorLogger(log.With().Str("component", "grpc_server").Logger())
	gsvr := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tags.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(interceptor, logOpts...),
		),
		grpc.ChainStreamInterceptor(
			tags.StreamServerInterceptor(),
			logging.StreamServerInterceptor(interceptor, logOpts...),
		),
	)
	server := api.NewServer(read, codec)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the shutdown.
	listener, err := net.Listen("tcp", flagAddress)
	if err != nil {
		log.Error().Str("address", flagAddress).Err(err).Msg("could not create listener")
		return failure
	}
	done := make(chan struct{})
	failed := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		follow.Run(ctx)
	}()
	go func() {
		start := time.Now()
		log.Info().Time("start", start).Msg("Flow DPS Live Indexer starting")
		err := fsm.Run()
		if err != nil {
			log.Warn().Err(err).Msg("Flow DPS Live Indexer failed")
			close(failed)
		} else {
			close(done)
		}
		finish := time.Now()
		duration := finish.Sub(start)
		log.Info().Time("finish", finish).Str("duration", duration.Round(time.Second).String()).Msg("Flow DPS Indexer stopped")
	}()
	go func() {
		log.Info().Msg("Flow DPS Live Server starting")
		api.RegisterAPIServer(gsvr, server)
		err = gsvr.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn().Err(err).Msg("Flow DPS Server failed")
		}
		log.Info().Msg("Flow DPS Live Server stopped")
	}()
	go func() {
		if !metricsEnabled {
			return
		}

		log.Info().Msg("metrics server starting")
		server := metrics.NewServer(log, flagMetrics)
		err := server.Start()
		if err != nil {
			log.Warn().Err(err).Msg("metrics server failed")
		}
		log.Info().Msg("metrics server stopped")
	}()

	// Here, we are waiting for a signal, or for one of the components to fail
	// or finish. In both cases, we proceed to shut down everything, while also
	// entering a goroutine that allows us to force shut down by sending
	// another signal.
	select {
	case <-sig:
		log.Info().Msg("Flow DPS Indexer stopping")
	case <-done:
		log.Info().Msg("Flow DPS Indexer done")
	case <-failed:
		log.Warn().Msg("Flow DPS Indexer aborted")
	}
	go func() {
		<-sig
		log.Warn().Msg("forcing exit")
		os.Exit(1)
	}()

	// We first stop serving the DPS API by shutting down the GRPC server. Next,
	// we shut down the consensus follower, so that there is no indexing to be
	// done anymore. Lastly, we stop the mapper logic itself.
	gsvr.GracefulStop()
	cancel()
	<-follow.Done()
	err = fsm.Stop()
	if err != nil {
		log.Error().Err(err).Msg("could not stop indexer")
		return failure
	}

	return success
}
