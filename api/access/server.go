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

package access

import (
	"context"
	"errors"
	"fmt"

	"github.com/c2h5oh/datasize"
	"github.com/dgraph-io/ristretto"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/engine/execution/state/delta"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/onflow/flow/protobuf/go/flow/entities"

	"github.com/optakt/flow-dps/models/index"
)

// Server is a simple implementation of the generated AccessAPIServer interface.
// It uses an index reader interface as the backend to retrieve the desired data.
// This is generally an on-disk interface, but could be a GRPC-based index as
// well, in which case there is a double redirection.
type Server struct {
	index index.Reader
	codec index.Codec

	vm    VirtualMachine
	cache Cache

	chainID string
}

// NewServer creates a new server, using the provided index reader as a backend
// for data retrieval.
func NewServer(index index.Reader, codec index.Codec, options ...func(*Config)) (*Server, error) {

	// Initialize the invoker configuration with conservative default values.
	cfg := Config{
		CacheSize: uint64(100 * datasize.MB), // ~100 MB default size
	}

	// Apply the option parameters provided by consumer.
	for _, option := range options {
		option(&cfg)
	}

	rt := fvm.NewInterpreterRuntime()
	vm := fvm.NewVirtualMachine(rt)

	// Initialize the Ristretto cache with the size limit. Ristretto recommends
	// keeping ten times as many counters as items in the cache when full.
	// Assuming an average item size of 1 kilobyte, this is what we get.
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(cfg.CacheSize) / 1000 * 10,
		MaxCost:     int64(cfg.CacheSize),
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize cache: %w", err)
	}

	s := Server{
		index: index,
		codec: codec,

		vm:    vm,
		cache: cache,

		chainID: cfg.ChainID,
	}

	return &s, nil
}

func (s *Server) Ping(_ context.Context, _ *access.PingRequest) (*access.PingResponse, error) {
	return &access.PingResponse{}, nil
}

func (s *Server) GetLatestBlockHeader(_ context.Context, _ *access.GetLatestBlockHeaderRequest) (*access.BlockHeaderResponse, error) {
	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block height: %w", err)
	}

	header, err := s.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block header: %w", err)
	}

	block, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("could not convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: block,
	}

	return &resp, err
}

func (s *Server) GetBlockHeaderByID(_ context.Context, in *access.GetBlockHeaderByIDRequest) (*access.BlockHeaderResponse, error) {
	blockID := flow.HashToID(in.Id)
	height, err := s.index.HeightForBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block height: %w", err)
	}

	header, err := s.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block header: %w", err)
	}

	block, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("could not convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: block,
	}

	return &resp, err
}

func (s *Server) GetBlockHeaderByHeight(_ context.Context, in *access.GetBlockHeaderByHeightRequest) (*access.BlockHeaderResponse, error) {
	header, err := s.index.Header(in.Height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block header: %w", err)
	}

	block, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("could not convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: block,
	}

	return &resp, err
}

func (s *Server) GetLatestBlock(ctx context.Context, in *access.GetLatestBlockRequest) (*access.BlockResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetBlockByID(ctx context.Context, in *access.GetBlockByIDRequest) (*access.BlockResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetBlockByHeight(ctx context.Context, in *access.GetBlockByHeightRequest) (*access.BlockResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetCollectionByID(_ context.Context, in *access.GetCollectionByIDRequest) (*access.CollectionResponse, error) {
	collId := flow.HashToID(in.Id)
	collection, err := s.index.Collection(collId)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve collection with ID %x: %w", in.Id, err)
	}

	collEntity := entities.Collection{
		Id: in.Id,
	}
	for _, txID := range collection.Transactions {
		collEntity.TransactionIds = append(collEntity.TransactionIds, txID[:])
	}

	resp := access.CollectionResponse{
		Collection: &collEntity,
	}

	return &resp, err
}

func (s *Server) SendTransaction(ctx context.Context, in *access.SendTransactionRequest) (*access.SendTransactionResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetTransaction(ctx context.Context, in *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetTransactionResult(ctx context.Context, in *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetAccount(_ context.Context, in *access.GetAccountRequest) (*access.GetAccountResponse, error) {
	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not get height: %w", err)
	}

	account, err := s.getAccount(flow.BytesToAddress(in.Address), height)
	if err != nil {
		return nil, err
	}

	// For now, we can't just reuse `GetAccountAtLatestBlock` for this because the return types are not the same.
	resp := access.GetAccountResponse{
		Account: account,
	}

	return &resp, nil
}

func (s *Server) GetAccountAtLatestBlock(ctx context.Context, in *access.GetAccountAtLatestBlockRequest) (*access.AccountResponse, error) {
	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not get height: %w", err)
	}

	// Simply call the height-specific endpoint with the latest height.
	req := &access.GetAccountAtBlockHeightRequest{
		Address:     in.Address,
		BlockHeight: height,
	}

	return s.GetAccountAtBlockHeight(ctx, req)
}

func (s *Server) GetAccountAtBlockHeight(_ context.Context, in *access.GetAccountAtBlockHeightRequest) (*access.AccountResponse, error) {
	account, err := s.getAccount(flow.BytesToAddress(in.Address), in.BlockHeight)
	if err != nil {
		return nil, err
	}

	resp := access.AccountResponse{
		Account: account,
	}

	return &resp, nil
}

func (s *Server) ExecuteScriptAtLatestBlock(ctx context.Context, in *access.ExecuteScriptAtLatestBlockRequest) (*access.ExecuteScriptResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) ExecuteScriptAtBlockID(ctx context.Context, in *access.ExecuteScriptAtBlockIDRequest) (*access.ExecuteScriptResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) ExecuteScriptAtBlockHeight(ctx context.Context, in *access.ExecuteScriptAtBlockHeightRequest) (*access.ExecuteScriptResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) GetEventsForHeightRange(_ context.Context, in *access.GetEventsForHeightRangeRequest) (*access.EventsResponse, error) {
	var events []*access.EventsResponse_Result
	for height := in.StartHeight; height <= in.EndHeight; height++ {
		ee, err := s.index.Events(height)
		if err != nil {
			return nil, fmt.Errorf("could not get events at height %d: %w", height, err)
		}

		header, err := s.index.Header(height)
		if err != nil {
			return nil, fmt.Errorf("could not get header at height %d: %w", height, err)
		}

		timestamp, err := ptypes.TimestampProto(header.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("could not parse timestamp for block at height %d: %w", height, err)
		}

		messages := make([]*entities.Event, 0, len(ee))
		for _, event := range ee {
			messages = append(messages, convert.EventToMessage(event))
		}

		blockID := header.ID()
		result := access.EventsResponse_Result{
			BlockId:        blockID[:],
			BlockHeight:    height,
			BlockTimestamp: timestamp,
			Events:         messages,
		}

		events = append(events, &result)
	}

	resp := access.EventsResponse{
		Results: events,
	}

	return &resp, nil
}

func (s *Server) GetEventsForBlockIDs(_ context.Context, in *access.GetEventsForBlockIDsRequest) (*access.EventsResponse, error) {
	var events []*access.EventsResponse_Result
	for _, id := range in.BlockIds {
		blockID := flow.HashToID(id)
		height, err := s.index.HeightForBlock(blockID)
		if err != nil {
			return nil, fmt.Errorf("could not get height of block with ID %x: %w", id, err)
		}

		ee, err := s.index.Events(height)
		if err != nil {
			return nil, fmt.Errorf("could not get events at height %d: %w", height, err)
		}

		header, err := s.index.Header(height)
		if err != nil {
			return nil, fmt.Errorf("could not get header at height %d: %w", height, err)
		}

		timestamp, err := ptypes.TimestampProto(header.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("could not parse timestamp for block at height %d: %w", height, err)
		}

		messages := make([]*entities.Event, 0, len(ee))
		for _, event := range ee {
			messages = append(messages, convert.EventToMessage(event))
		}

		result := access.EventsResponse_Result{
			BlockId:        blockID[:],
			BlockHeight:    height,
			BlockTimestamp: timestamp,
			Events:         messages,
		}

		events = append(events, &result)
	}

	resp := access.EventsResponse{
		Results: events,
	}

	return &resp, nil
}

func (s *Server) GetNetworkParameters(_ context.Context, _ *access.GetNetworkParametersRequest) (*access.GetNetworkParametersResponse, error) {
	root, err := s.index.First()
	if err != nil {
		return nil, fmt.Errorf("could not get first indexed height: %w", err)
	}

	header, err := s.index.Header(root)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	return &access.GetNetworkParametersResponse{ChainId: header.ChainID.String()}, nil
}

func (s *Server) GetLatestProtocolStateSnapshot(ctx context.Context, in *access.GetLatestProtocolStateSnapshotRequest) (*access.ProtocolStateSnapshotResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *Server) getAccount(address flow.Address, height uint64) (*entities.Account, error) {
	header, err := s.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header at height %d: %w", height, err)
	}

	ctx := fvm.NewContext(zerolog.Nop(), fvm.WithBlockHeader(header))

	// Initialize the read function. We use a shared cache between all heights
	// here. It's a smart cache, which means that items that are accessed often
	// are more likely to be kept, regardless of height. This allows us to put
	// an upper bound on total cache size while using it for all heights.
	read := readRegister(s.index, s.cache, height)

	// Initialize the view of the execution state on top of the ledger by
	// using the read function at a specific commit.
	view := delta.NewView(read)

	account, err := s.vm.GetAccount(ctx, address, view, programs.NewEmptyPrograms())
	if err != nil {
		return nil, fmt.Errorf("could not get account at height %d: %w", height, err)
	}

	a, err := convert.AccountToMessage(account)
	if err != nil {
		return nil, fmt.Errorf("could not convert account to message: %w", err)
	}

	return a, nil
}
