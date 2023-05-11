package main

/*

This is a simple ledger-heavy benchmark that can be run against devnet's access API implementation.

Example usage:
$ go run main.go --address 35.208.135.180:9000 --start-height 99445069  --end-height 99452069
$ go run main.go --address 35.208.111.49:9000  --start-height 100675453 --end-height 100775424
*/

import (
	"context"
	"flag"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	_ "embed"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/access/grpc"
)

//go:embed get_total_balance.cdc
var script []byte

const (
	defaultMaxConcurrent = 100
	defaultLoopCount     = 100000
)

var (
	maxConcurrent int
	batchSize     int
	loopCount     int
	startHeight   int64
	endHeight     int64

	address        string
	metricsAddress string
)

var (
	latencyHist = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "flow_script_execution_latency_ms",
		Help:    "Latency of script execution",
		Buckets: prometheus.ExponentialBucketsRange(1, 30000, 30),
	})
)

type BatchAddressGenerator struct {
	sync.Mutex

	gen       *flow.AddressGenerator
	batchSize int
}

func NewBatchAddressGenerator(network flow.ChainID, batchSize int) *BatchAddressGenerator {
	return &BatchAddressGenerator{
		gen:       flow.NewAddressGenerator(network),
		batchSize: batchSize,
	}
}

func (g *BatchAddressGenerator) genNextAddress() flow.Address {
	g.Lock()
	defer g.Unlock()

	return g.gen.NextAddress()
}

func (g *BatchAddressGenerator) NextBatch() []cadence.Value {
	addresses := make([]cadence.Value, g.batchSize)
	for i := 0; i < g.batchSize; i++ {
		nextAddressBytes := g.genNextAddress().Bytes()
		addresses[i] = cadence.NewAddress(([8]byte)(nextAddressBytes))
	}
	return addresses
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flag.IntVar(&maxConcurrent, "max-concurrent", defaultMaxConcurrent, "max concurrent requests")
	flag.IntVar(&loopCount, "loop-count", defaultLoopCount, "number of requests to make")
	flag.Int64Var(&startHeight, "start-height", 0, "start height")
	flag.Int64Var(&endHeight, "end-height", 0, "end height")
	flag.StringVar(&address, "address", "localhost:3569", "host:port of the flow access API (grpc) server")
	flag.StringVar(&metricsAddress, "metrics-address", "localhost:0", "host:port of the metrics server")
	flag.IntVar(&batchSize, "batch-size", 100, "number of addresses to generate per batch")

	flag.Parse()

	if startHeight == 0 || endHeight == 0 {
		log.Fatal().Msg("start and end height must be set")
	}

	ctx := context.Background()

	c, err := grpc.NewClient(address)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create grpc client")
	}

	go func() {
		metricsListener, err := net.Listen("tcp", metricsAddress)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to listen for metrics")
		}

		log.Info().Str("address", metricsListener.Addr().String()).Msg("metrics server listening")
		http.Handle("/metrics", promhttp.Handler())
		http.Serve(metricsListener, nil)
	}()

	totalTime := atomic.NewUint64(0)
	totalCount := atomic.NewUint64(0)

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(maxConcurrent)

	generator := NewBatchAddressGenerator(flow.Testnet, batchSize)
	for loop := 0; loop < loopCount; loop++ {
		eg.Go(func() error {
			scriptStart := time.Now()

			arguments := []cadence.Value{
				cadence.NewArray(generator.NextBatch()),
			}
			_, err := c.ExecuteScriptAtBlockHeight(ctx, uint64(rand.Int63n(endHeight-startHeight)+startHeight), script, arguments)
			if err != nil {
				log.Error().Err(err).Msg("failed to execute script")
				return nil
			}
			latency := time.Since(scriptStart)
			latencyHist.Observe(float64(latency.Milliseconds()))

			totalCount.Add(1)
			totalTime.Add(uint64(latency.Nanoseconds()))

			return nil
		})

		if (loop > 100 && loop%100 == 0) || loop == loopCount-1 {
			currentCount := totalCount.Load()
			if currentCount == 0 {
				continue
			}

			log.Info().
				Dur("ms_per_call", time.Duration(totalTime.Load()/currentCount)).
				Msg("progress")
		}
	}

	err = eg.Wait()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to wait for all requests")
	}
}
