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

package dps

import (
	"context"
	"fmt"

	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/optakt/flow-dps/models/index"
)

// Server is a simple implementation of the generated AccessAPIServer interface.
// It uses an index reader interface as the backend to retrieve the desired data.
// This is generally an on-disk interface, but could be a GRPC-based index as
// well, in which case there is a double redirection.
type Server struct {
	index index.Reader
	codec index.Codec
}

// NewServer creates a new server, using the provided index reader as a backend
// for data retrieval.
func NewServer(index index.Reader, codec index.Codec) *Server {

	s := Server{
		index: index,
		codec: codec,
	}

	return &s
}

func (s *Server) Ping(ctx context.Context, in *access.PingRequest) (*access.PingResponse, error) {
	panic("implement me")
}

func (s *Server) GetLatestBlockHeader(_ context.Context, _ *access.GetLatestBlockHeaderRequest) (*access.BlockHeaderResponse, error) {
	lastHeight, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve last block height: %w", err)
	}

	header, err := s.index.Header(lastHeight)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve last block header: %w", err)
	}

	block, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("unable to convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: block,
	}

	return &resp, err
}

func (s *Server) GetBlockHeaderByID(_ context.Context, in *access.GetBlockHeaderByIDRequest) (*access.BlockHeaderResponse, error) {
	var blockID flow.Identifier
	copy(blockID[:], in.Id)

	blockHeight, err := s.index.HeightForBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve last block height: %w", err)
	}

	header, err := s.index.Header(blockHeight)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve last block header: %w", err)
	}

	block, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("unable to convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: block,
	}

	return &resp, err
}

func (s *Server) GetBlockHeaderByHeight(_ context.Context, in *access.GetBlockHeaderByHeightRequest) (*access.BlockHeaderResponse, error) {
	header, err := s.index.Header(in.Height)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve last block header: %w", err)
	}

	block, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("unable to convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: block,
	}

	return &resp, err
}

func (s *Server) GetLatestBlock(ctx context.Context, in *access.GetLatestBlockRequest) (*access.BlockResponse, error) {
	panic("implement me")
}

func (s *Server) GetBlockByID(ctx context.Context, in *access.GetBlockByIDRequest) (*access.BlockResponse, error) {
	panic("implement me")
}

func (s *Server) GetBlockByHeight(ctx context.Context, in *access.GetBlockByHeightRequest) (*access.BlockResponse, error) {
	panic("implement me")
}

func (s *Server) GetCollectionByID(ctx context.Context, in *access.GetCollectionByIDRequest) (*access.CollectionResponse, error) {
	panic("implement me")
}

func (s *Server) SendTransaction(ctx context.Context, in *access.SendTransactionRequest) (*access.SendTransactionResponse, error) {
	panic("implement me")
}

func (s *Server) GetTransaction(ctx context.Context, in *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	panic("implement me")
}

func (s *Server) GetTransactionResult(ctx context.Context, in *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	panic("implement me")
}

func (s *Server) GetAccount(ctx context.Context, in *access.GetAccountRequest) (*access.GetAccountResponse, error) {
	panic("implement me")
}

func (s *Server) GetAccountAtLatestBlock(ctx context.Context, in *access.GetAccountAtLatestBlockRequest) (*access.AccountResponse, error) {
	panic("implement me")
}

func (s *Server) GetAccountAtBlockHeight(ctx context.Context, in *access.GetAccountAtBlockHeightRequest) (*access.AccountResponse, error) {
	panic("implement me")
}

func (s *Server) ExecuteScriptAtLatestBlock(ctx context.Context, in *access.ExecuteScriptAtLatestBlockRequest) (*access.ExecuteScriptResponse, error) {
	panic("implement me")
}

func (s *Server) ExecuteScriptAtBlockID(ctx context.Context, in *access.ExecuteScriptAtBlockIDRequest) (*access.ExecuteScriptResponse, error) {
	panic("implement me")
}

func (s *Server) ExecuteScriptAtBlockHeight(ctx context.Context, in *access.ExecuteScriptAtBlockHeightRequest) (*access.ExecuteScriptResponse, error) {
	panic("implement me")
}

func (s *Server) GetEventsForHeightRange(ctx context.Context, in *access.GetEventsForHeightRangeRequest) (*access.EventsResponse, error) {
	panic("implement me")
}

func (s *Server) GetEventsForBlockIDs(ctx context.Context, in *access.GetEventsForBlockIDsRequest) (*access.EventsResponse, error) {
	panic("implement me")
}

func (s *Server) GetNetworkParameters(ctx context.Context, in *access.GetNetworkParametersRequest) (*access.GetNetworkParametersResponse, error) {
	panic("implement me")
}

func (s *Server) GetLatestProtocolStateSnapshot(ctx context.Context, in *access.GetLatestProtocolStateSnapshotRequest) (*access.ProtocolStateSnapshotResponse, error) {
	panic("implement me")
}
