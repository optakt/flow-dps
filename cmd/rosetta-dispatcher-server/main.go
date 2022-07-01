package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/ziflex/lecho/v2"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/labstack/echo/v4/middleware"
	"github.com/optakt/flow-dps-rosetta/service/identifier"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

const (
	success = 0
	failure = 1
)

type Spork struct {
	First      uint64
	Last       uint64
	ProxyTaget *middleware.ProxyTarget
}

type SporkList []Spork

func NewSporkList(addresses []string, firsts []int64, lasts []int64) (SporkList, error) {
	// TODO: Add tests for validating sporks configuration

	// Check past sporks configuration
	if len(addresses) != len(firsts) || len(addresses) != len(lasts) {
		return nil, fmt.Errorf("data length mismatch")
	}

	if len(addresses) < 2 {
		return nil, fmt.Errorf("at least two sporks must be provided")
	}

	sporksTable := make(SporkList, len(addresses))

	for i, sporkAddress := range addresses {
		sporksTable[i].First = uint64(firsts[i])
		sporksTable[i].Last = uint64(lasts[i])

		url, err := url2.Parse(sporkAddress)
		if err != nil {
			return nil, fmt.Errorf("spork address %d is invalid: %w", i, err)
		}
		if sporksTable[i].First >= sporksTable[i].Last {
			return nil, fmt.Errorf("spork %d last height is not greater than first", i)
		}

		sporksTable[i].ProxyTaget = &middleware.ProxyTarget{URL: url}
	}

	sort.Sort(sporksTable)

	for i := 0; i < len(sporksTable)-1; i++ {
		if sporksTable[i].Last != sporksTable[i+1].First-1 {
			return nil, fmt.Errorf("gap between sporks boundaries: %d and %d", sporksTable[i].Last, sporksTable[i+1].First)
		}
	}

	return sporksTable, nil
}

func (s SporkList) Len() int {
	return len(s)
}

func (s SporkList) Less(i, j int) bool {
	return s[i].First < s[j].First
}

func (s SporkList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SporkList) ServerForHeight(height uint64) (*middleware.ProxyTarget, error) {
	// Assume sorted and more than zero items

	first := s[0].First
	last := s[len(s)-1].Last

	if height < first {
		return nil, fmt.Errorf("height %d below supported range %d - %d", height, first, last)
	}

	if height > last {
		return nil, fmt.Errorf("height %d above supported range %d - %d", height, first, last)
	}

	for _, spork := range s {
		if height >= spork.First && height <= spork.Last {
			return spork.ProxyTaget, nil
		}
	}

	return nil, fmt.Errorf("spork for height %d not found", height)
}

func NewFlowHeightAwareBalancer(sporkList SporkList) *FlowHeightAwareBalancer {

	//targets := make([]*middleware.ProxyTarget, sporkList.Len())
	//
	//for i, spork := range sporkList {
	//
	//}

	return &FlowHeightAwareBalancer{
		//targets:   nil,
		sporkList: sporkList,
	}
}

type FlowHeightAwareBalancer struct {
	//targets []*middleware.ProxyTarget
	//mutex   sync.RWMutex
	sporkList SporkList
}

func (f *FlowHeightAwareBalancer) AddTarget(target *middleware.ProxyTarget) bool {
	// boilerplate from labstack/echi
	//for _, t := range f.targets {
	//	if t.Name == target.Name {
	//		return false
	//	}
	//}
	//f.mutex.Lock()
	//defer f.mutex.Unlock()
	//f.targets = append(f.targets, target)
	//return true
	return false
}

func (f *FlowHeightAwareBalancer) RemoveTarget(name string) bool {
	//f.mutex.Lock()
	//defer f.mutex.Unlock()
	//for i, t := range f.targets {
	//	if t.Name == name {
	//		f.targets = append(f.targets[:i], f.targets[i+1:]...)
	//		return true
	//	}
	//}
	//return false
	return false
}

// All requests which  have BlockID have them in the same field
type BlockAwareRequest struct {
	BlockID identifier.Block `json:"block_identifier,omitempty"`
}

func (f *FlowHeightAwareBalancer) Next(e echo.Context) *middleware.ProxyTarget {
	blockAware := BlockAwareRequest{}

	var reqBody []byte
	if e.Request().Body != nil { // Read
		reqBody, _ = ioutil.ReadAll(e.Request().Body)
	}
	e.Request().Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // Reset

	err := json.Unmarshal(reqBody, &blockAware)
	if err != nil {
		e.Error(err)
	}

	// no block ID = just use latest spork
	// TODO - or maybe random?
	blockHeight := uint64(0)
	if blockAware.BlockID.Index != nil {
		blockHeight = *blockAware.BlockID.Index
	}
	if blockHeight == 0 {
		return f.sporkList[len(f.sporkList)-1].ProxyTaget
	}

	for _, spork := range f.sporkList {
		if blockHeight >= spork.First && blockHeight <= spork.Last {
			return spork.ProxyTaget
		}
	}

	// return last if nothing matches
	return f.sporkList[len(f.sporkList)-1].ProxyTaget
}

func main() {
	os.Exit(run())
}

func run() int {

	// Signal catching for clean shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// Command line parameter initialization.
	var (
		flagLevel string
		flagPort  uint16

		flagSporkAddresses []string
		flagSporkFirsts    []int64
		flagSporkLast      []int64
	)

	pflag.StringVarP(&flagLevel, "level", "l", "info", "log output level")
	pflag.Uint16VarP(&flagPort, "port", "p", 8080, "port to host Rosetta API on")

	pflag.StringSliceVar(&flagSporkAddresses, "spork-addresses", nil, "comma-separated list of past sporks Rosetta API servers")
	pflag.Int64SliceVar(&flagSporkFirsts, "spork-firsts", nil, "comma-separated list of past sporks first supported block height, corresponding to spork addresses")
	pflag.Int64SliceVar(&flagSporkLast, "spork-lasts", nil, "comma-separated list of past sporks last supported block height, corresponding to spork addresses")

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
	elog := lecho.From(log)

	sporkList, err := NewSporkList(flagSporkAddresses, flagSporkFirsts, flagSporkLast)
	if err != nil {
		log.Error().Err(err).Msg("spork list configuration error")
		return failure
	}

	server := echo.New()
	server.HideBanner = true
	server.HidePort = true
	server.Logger = elog
	server.Use(lecho.Middleware(lecho.Config{Logger: elog}))
	server.Use(middleware.Proxy(NewFlowHeightAwareBalancer(sporkList)))

	// This section launches the main executing components in their own
	// goroutine, so they can run concurrently. Afterwards, we wait for an
	// interrupt signal in order to proceed with the next section.
	done := make(chan struct{})
	failed := make(chan struct{})
	go func() {
		log.Info().Msg("Rosetta Dispatcher Server starting")
		err := server.Start(fmt.Sprint(":", flagPort))
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warn().Err(err).Msg("Rosetta Dispatcher Server failed")
			close(failed)
		} else {
			close(done)
		}
		log.Info().Msg("Rosetta Dispatcher Server stopped")
	}()

	select {
	case <-sig:
		log.Info().Msg("Rosetta Dispatcher Server stopping")
	case <-done:
		log.Info().Msg("Rosetta Dispatcher Server done")
	case <-failed:
		log.Warn().Msg("Rosetta Dispatcher Server aborted")
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	if err != nil {
		log.Error().Err(err).Msg("could not shut down Rosetta Dispatcher Server")
		return failure
	}

	return success
}
