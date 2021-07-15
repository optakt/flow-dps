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

	"github.com/golang/protobuf/ptypes"

	"github.com/onflow/flow-go/engine/common/rpc/convert"
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
	index   index.Reader
	codec   index.Codec
	invoker Invoker

	chainID string
}

// NewServer creates a new server, using the provided index reader as a backend
// for data retrieval.
func NewServer(index index.Reader, codec index.Codec, invoker Invoker, chainID string) *Server {
	s := Server{
		index:   index,
		codec:   codec,
		invoker: invoker,

		chainID: chainID,
	}

	return &s
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

func (s *Server) GetTransaction(_ context.Context, in *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	id := flow.HashToID(in.Id)
	tb, err := s.index.Transaction(id)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction: %w", err)
	}

	resp := access.TransactionResponse{
		Transaction: convert.TransactionToMessage(*tb),
	}

	return &resp, nil
}

func (s *Server) GetTransactionResult(_ context.Context, in *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	txID := flow.HashToID(in.Id)
	tx, err := s.index.Transaction(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction: %w", err)
	}

	result, err := s.index.Result(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction result: %w", err)
	}

	// The code below is a simplified and limited implementation to derive the transaction status from
	// https://github.com/onflow/flow-go/blob/v0.17.4/engine/access/rpc/backend/backend_transactions.go#L257-L328
	statusCode := uint32(0)
	if result.ErrorMessage == "" {
		statusCode = 1
	}

	// First, we need to retrieve the last indexed height to access the latest seals.
	lastHeight, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last height: %w", err)
	}

	sealIDs, err := s.index.SealsByHeight(lastHeight)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block seals: %w", err)
	}

	// We then go through all sealed blocks at the last height and look for the latest sealed height.
	var sealedHeight uint64
	for _, sealID := range sealIDs {
		seal, err := s.index.Seal(sealID)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve seal: %w", err)
		}

		h, err := s.index.HeightForBlock(seal.BlockID)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve block height: %w", err)
		}

		if h > sealedHeight {
			sealedHeight = h
		}
	}

	// We also need the height of the transaction we're looking at.
	height, err := s.index.HeightForBlock(tx.ReferenceBlockID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block height: %w", err)
	}

	// Finally, if the transaction occurred after the current sealed height, we can say it's executed,
	// otherwise that it is sealed. Pending, finalized and expired transactions are not supported by our
	// API since we do not have access to pending and expired ones, and that finalized ones are also executed
	// on our side, which takes precedence according to the aforementioned implementation in Flow-Go.
	status := entities.TransactionStatus_SEALED
	if height > sealedHeight {
		status = entities.TransactionStatus_EXECUTED
	}

	events, err := s.index.Events(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	resp := access.TransactionResultResponse{
		Status:       status,
		StatusCode:   statusCode,
		ErrorMessage: result.ErrorMessage,
		Events:       convert.EventsToMessages(events),
		BlockId:      tx.ReferenceBlockID[:],
	}

	return &resp, nil
}

func (s *Server) GetAccount(ctx context.Context, in *access.GetAccountRequest) (*access.GetAccountResponse, error) {
	req := access.GetAccountAtLatestBlockRequest{
		Address: in.Address,
	}

	account, err := s.GetAccountAtLatestBlock(ctx, &req)
	if err != nil {
		return nil, err
	}

	resp := access.GetAccountResponse{
		Account: account.Account,
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
	header, err := s.index.Header(in.BlockHeight)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	account, err := s.invoker.GetAccount(flow.BytesToAddress(in.Address), header)
	if err != nil {
		return nil, err
	}

	a, err := convert.AccountToMessage(account)
	if err != nil {
		return nil, fmt.Errorf("could not convert account to RPC message: %w", err)
	}

	resp := access.AccountResponse{
		Account: a,
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
