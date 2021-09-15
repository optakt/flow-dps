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

	api "github.com/optakt/flow-dps/api/dps"
	"github.com/optakt/flow-dps/codec/zbor"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/service/cloud"
	"github.com/optakt/flow-dps/service/forest"
	"github.com/optakt/flow-dps/service/index"
	"github.com/optakt/flow-dps/service/initializer"
	"github.com/optakt/flow-dps/service/mapper"
	"github.com/optakt/flow-dps/service/storage"
	"github.com/optakt/flow-dps/service/tracker"
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
		flagForce      bool
		flagIndex      string
		flagLevel      string
		flagSkip       bool

		flagSeedAddress string
		flagSeedKey     string
	)

	pflag.StringVarP(&flagAddress, "address", "a", "127.0.0.1:5005", "address to serve the GRPC DPS API on")
	pflag.StringVarP(&flagBootstrap, "bootstrap", "b", "bootstrap", "path to directory with public bootstrap information for the spork")
	pflag.StringVarP(&flagBucket, "bucket", "u", "", "name of the Google Cloud Storage bucket which contains the block data")
	pflag.StringVarP(&flagCheckpoint, "checkpoint", "c", "root.checkpoint", "checkpoint file for state trie")
	pflag.StringVarP(&flagData, "data", "d", "data", "database directory for protocol data")
	pflag.BoolVarP(&flagForce, "force", "f", false, "overwrite existing index database")
	pflag.StringVarP(&flagIndex, "index", "i", "index", "database directory for state index")
	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.BoolVarP(&flagSkip, "skip", "s", false, "skip indexing of execution state ledger registers")

	pflag.StringVar(&flagSeedAddress, "seed-address", "", "address of the seed node to follow unstaked consensus")
	pflag.StringVar(&flagSeedKey, "seed-key", "", "hex-encoded public network key of the seed node to follow unstaked consensus")

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

	// Open DPS index database.
	db, err := badger.Open(dps.DefaultOptions(flagIndex))
	if err != nil {
		log.Error().Str("index", flagIndex).Err(err).Msg("could not open index database")
		return failure
	}
	defer func() {
		err := db.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index database")
		}
	}()

	// Open protocol state database.
	data, err := badger.Open(dps.DefaultOptions(flagData))
	if err != nil {
		log.Error().Err(err).Msg("could not open protocol state database")
		return failure
	}
	defer func() {
		err := data.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close protocol state database")
		}
	}()

	// The storage library is initialized with a codec and provides functions to
	// interact with a Badger database while encoding and compressing
	// transparently.
	codec := zbor.NewCodec()
	storage := storage.New(codec)

	// Initialize the index reader and check whether there is already an index
	// in the database at the provided index database directory.
	read := index.NewReader(db, storage)
	_, err = read.First()
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		log.Error().Err(err).Msg("could not get first height from index reader")
		return failure
	}
	indexExists := err == nil
	if indexExists && !flagForce {
		log.Error().Err(err).Msg("index already exists, manually delete it or use (-f, --force) to overwrite it")
		return failure
	}

	// Unfortunately, the current consensus follower does not bootstrap the
	// protocol state until it is run. This means we are liable to miss block
	// finalization of the first few blocks, which breaks our logic. We thus
	// must ensure that the protocol state is already bootstrapped manually
	// before starting the consensus follower.
	path := filepath.Join(flagBootstrap, bootstrap.PathRootProtocolStateSnapshot)
	file, err := os.Open(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("could not open protocol state snapshot")
		return failure
	}
	err = initializer.ProtocolState(file, data)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize protocol state")
		return failure
	}

	// Initialize the private key for joining the unstaked peer-to-peer network.
	// This is just needed for security, not authentication, so we can just
	// generate a new one each time we start.
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

	// Initialize the unstaked consensus follower. It connects to a staked
	// access node for bootstrapping the peer-to-peer network with other
	// staked access nodes and unstaked consensus followers. For every finalized
	// block, it calls the provided callback, which lets the DPS consensus
	// follower update its data.
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
		unstaked.WithDB(data),
		unstaked.WithLogLevel(flagLevel),
	)
	if err != nil {
		log.Error().Err(err).Str("bucket", flagBucket).Msg("could not create consensus follower")
		return failure
	}

	// Initialize the execution follower that will read block records from the
	// Google Cloud Platform bucket.
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
	stream := cloud.NewGCPStreamer(log, bucket)
	execution, err := tracker.NewExecution(log, data, stream)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize execution tracker")
		return failure
	}

	// Initialize the consensus tracker, which uses the protocol state to
	// retrieve data from consensus and the execution follower to complement
	// the data with complete blocks.
	consensus, err := tracker.NewConsensus(log, data, execution)
	if err != nil {
		log.Error().Err(err).Msg("could not initialize consensus tracker")
		return failure
	}

	// We can now register both the consensus follower and the cloud streamer
	// as consumers of finalized blocks.
	follow.AddOnBlockFinalizedConsumer(stream.OnBlockFinalized)
	follow.AddOnBlockFinalizedConsumer(consensus.OnBlockFinalized)

	// Initialize the index writer, which is responsible for writing the chain
	// and execution data to the index database.
	write := index.NewWriter(db, storage)
	defer func() {
		err := write.Close()
		if err != nil {
			log.Error().Err(err).Msg("could not close index writer")
		}
	}()

	// Load the root trie from the checkpoint file providen on command line.
	root, err := initializer.RootTrie(flagCheckpoint)
	if err != nil {
		log.Error().Err(err).Msg("could not load root checkpoint")
		return failure
	}

	// Initialize the state transition library, the finite-state machine (FSM)
	// and then register the desired state transitions with the FSM.
	transitions := mapper.NewTransitions(log, root, consensus, execution, write,
		mapper.WithSkipRegisters(flagSkip),
	)
	forest := forest.New()
	state := mapper.EmptyState(forest)
	fsm := mapper.NewFSM(state,
		mapper.WithTransition(mapper.StatusBootstrap, transitions.BootstrapState),
		mapper.WithTransition(mapper.StatusIndex, transitions.IndexChain),
		mapper.WithTransition(mapper.StatusUpdate, transitions.UpdateTree),
		mapper.WithTransition(mapper.StatusCollect, transitions.CollectRegisters),
		mapper.WithTransition(mapper.StatusMap, transitions.MapRegisters),
		mapper.WithTransition(mapper.StatusForward, transitions.ForwardHeight),
	)

	// Initialize the GRPC server for the DPS API.
	opts := []logging.Option{
		logging.WithLevels(logging.DefaultServerCodeToLevel),
	}
	interceptor := grpczerolog.InterceptorLogger(log.With().Str("component", "grpc_server").Logger())
	gsvr := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tags.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(interceptor, opts...),
		),
		grpc.ChainStreamInterceptor(
			tags.StreamServerInterceptor(),
			logging.StreamServerInterceptor(interceptor, opts...),
		),
	)
	server := api.NewServer(read, codec)

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
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
			close(failed)
		} else {
			close(done)
		}
		log.Info().Msg("Flow DPS Live Server stopped")
	}()

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

	// First, stop the DPS API to avoid failed requests.
	gsvr.GracefulStop()

	// Then stop the consensus follower and wait for it to shut down completely.
	cancel()
	<-follow.NodeBuilder.Done()

	// Lastly, we can stop the core business logic of the indexer.
	err = fsm.Stop()
	if err != nil {
		log.Error().Err(err).Msg("could not stop indexer")
		return failure
	}

	return success
}
