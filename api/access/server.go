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
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/onflow/flow/protobuf/go/flow/entities"

	"github.com/optakt/flow-dps/models/dps"
)

// Server is a simple implementation of the generated AccessAPIServer interface.
// It uses an index reader interface as the backend to retrieve the desired data.
// This is generally an on-disk interface, but could be a GRPC-based index as
// well, in which case there is a double redirection.
type Server struct {
	index   dps.Reader
	codec   dps.Codec
	invoker Invoker
}

// NewServer creates a new server, using the provided index reader as a backend
// for data retrieval.
func NewServer(index dps.Reader, codec dps.Codec, invoker Invoker) *Server {
	s := Server{
		index:   index,
		codec:   codec,
		invoker: invoker,
	}

	return &s
}

func (s *Server) Ping(_ context.Context, _ *access.PingRequest) (*access.PingResponse, error) {
	return &access.PingResponse{}, nil
}

func (s *Server) GetLatestBlockHeader(ctx context.Context, _ *access.GetLatestBlockHeaderRequest) (*access.BlockHeaderResponse, error) {
	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block height: %w", err)
	}

	req := access.GetBlockHeaderByHeightRequest{
		Height: height,
	}

	return s.GetBlockHeaderByHeight(ctx, &req)
}

func (s *Server) GetBlockHeaderByID(ctx context.Context, in *access.GetBlockHeaderByIDRequest) (*access.BlockHeaderResponse, error) {
	blockID := flow.HashToID(in.Id)
	height, err := s.index.HeightForBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block height: %w", err)
	}

	req := access.GetBlockHeaderByHeightRequest{
		Height: height,
	}

	return s.GetBlockHeaderByHeight(ctx, &req)
}

func (s *Server) GetBlockHeaderByHeight(_ context.Context, in *access.GetBlockHeaderByHeightRequest) (*access.BlockHeaderResponse, error) {
	header, err := s.index.Header(in.Height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last block header: %w", err)
	}

	blockMsg, err := convert.BlockHeaderToMessage(header)
	if err != nil {
		return nil, fmt.Errorf("could not convert block header to RPC entity: %w", err)
	}

	resp := access.BlockHeaderResponse{
		Block: blockMsg,
	}

	return &resp, err
}

func (s *Server) GetLatestBlock(ctx context.Context, in *access.GetLatestBlockRequest) (*access.BlockResponse, error) {
	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not get last height: %w", err)
	}

	req := &access.GetBlockByHeightRequest{
		Height: height,
	}

	return s.GetBlockByHeight(ctx, req)
}

func (s *Server) GetBlockByID(ctx context.Context, in *access.GetBlockByIDRequest) (*access.BlockResponse, error) {
	blockID := flow.HashToID(in.Id)
	height, err := s.index.HeightForBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get height for block %x: %w", blockID, err)
	}

	req := access.GetBlockByHeightRequest{
		Height: height,
	}

	return s.GetBlockByHeight(ctx, &req)
}

func (s *Server) GetBlockByHeight(_ context.Context, in *access.GetBlockByHeightRequest) (*access.BlockResponse, error) {
	header, err := s.index.Header(in.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get header for height %d: %w", in.Height, err)
	}

	sealIDs, err := s.index.SealsByHeight(in.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get seals for height %d: %w", in.Height, err)
	}

	seals := make([]*entities.BlockSeal, 0, len(sealIDs))
	for _, sealID := range sealIDs {
		seal, err := s.index.Seal(sealID)
		if err != nil {
			return nil, fmt.Errorf("could not get seal from ID %x: %w", sealID, err)
		}

		// See https://github.com/onflow/flow-go/blob/v0.17.4/engine/common/rpc/convert/convert.go#L180-L188
		entity := entities.BlockSeal{
			BlockId:                    seal.BlockID[:],
			ExecutionReceiptId:         seal.ResultID[:],
			ExecutionReceiptSignatures: [][]byte{}, // filling seals signature with zero
		}
		seals = append(seals, &entity)
	}

	collIDs, err := s.index.CollectionsByHeight(in.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get collections for height %d: %w", in.Height, err)
	}

	collections := make([]*entities.CollectionGuarantee, 0, len(collIDs))
	for _, collID := range collIDs {
		guarantee, err := s.index.Guarantee(collID)
		if err != nil {
			return nil, fmt.Errorf("could not get collection from ID %x: %w", collID, err)
		}

		entity := entities.CollectionGuarantee{
			CollectionId: collID[:],
			Signatures:   [][]byte{guarantee.Signature},
		}
		collections = append(collections, &entity)
	}

	blockID := header.ID()
	block := entities.Block{
		Id:                   blockID[:],
		Height:               in.Height,
		ParentId:             header.ParentID[:],
		Timestamp:            timestamppb.New(header.Timestamp),
		CollectionGuarantees: collections,
		BlockSeals:           seals,
		Signatures:           [][]byte{header.ParentVoterSig},
	}

	resp := access.BlockResponse{
		Block: &block,
	}

	return &resp, nil
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

func (s *Server) GetTransaction(_ context.Context, in *access.GetTransactionRequest) (*access.TransactionResponse, error) {
	txID := flow.HashToID(in.Id)
	tx, err := s.index.Transaction(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction: %w", err)
	}

	resp := access.TransactionResponse{
		Transaction: convert.TransactionToMessage(*tx),
	}

	return &resp, nil
}

func (s *Server) GetTransactionResult(_ context.Context, in *access.GetTransactionRequest) (*access.TransactionResultResponse, error) {
	txID := flow.HashToID(in.Id)
	result, err := s.index.Result(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction result: %w", err)
	}

	// We also need the height of the transaction we're looking at.
	// TODO: See if it wouldn't make more sense to remove this index and just
	//		 always return the status as sealed.
	//		 https://github.com/optakt/flow-dps/issues/317
	height, err := s.index.HeightForTransaction(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block height: %w", err)
	}

	block, err := s.index.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block header: %w", err)
	}
	blockID := block.ID()

	statusCode := uint32(0)
	if result.ErrorMessage == "" {
		statusCode = 1
	}

	status := entities.TransactionStatus_SEALED
	sealedHeight, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve last height: %w", err)
	}

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
		BlockId:      blockID[:],
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

	accountMsg, err := convert.AccountToMessage(account)
	if err != nil {
		return nil, fmt.Errorf("could not convert account to RPC message: %w", err)
	}

	resp := access.AccountResponse{
		Account: accountMsg,
	}

	return &resp, nil
}

func (s *Server) ExecuteScriptAtLatestBlock(ctx context.Context, in *access.ExecuteScriptAtLatestBlockRequest) (*access.ExecuteScriptResponse, error) {
	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not get last height: %w", err)
	}

	req := &access.ExecuteScriptAtBlockHeightRequest{
		BlockHeight: height,
		Script:      in.Script,
		Arguments:   in.Arguments,
	}

	return s.ExecuteScriptAtBlockHeight(ctx, req)
}

func (s *Server) ExecuteScriptAtBlockID(ctx context.Context, in *access.ExecuteScriptAtBlockIDRequest) (*access.ExecuteScriptResponse, error) {
	blockID := flow.HashToID(in.BlockId)
	height, err := s.index.HeightForBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get height for block ID %x: %w", blockID, err)
	}

	req := &access.ExecuteScriptAtBlockHeightRequest{
		BlockHeight: height,
		Script:      in.Script,
		Arguments:   in.Arguments,
	}

	return s.ExecuteScriptAtBlockHeight(ctx, req)
}

func (s *Server) ExecuteScriptAtBlockHeight(_ context.Context, in *access.ExecuteScriptAtBlockHeightRequest) (*access.ExecuteScriptResponse, error) {
	var args []cadence.Value
	for _, arg := range in.Arguments {
		val, err := json.Decode(arg)
		if err != nil {
			return nil, fmt.Errorf("could not decode script argument: %w", err)
		}

		args = append(args, val)
	}

	value, err := s.invoker.Script(in.BlockHeight, in.Script, args)
	if err != nil {
		return nil, fmt.Errorf("could not execute script: %w", err)
	}

	result, err := json.Encode(value)
	if err != nil {
		return nil, fmt.Errorf("could not encode script result: %w", err)
	}

	resp := access.ExecuteScriptResponse{
		Value: result,
	}

	return &resp, nil
}

func (s *Server) GetEventsForHeightRange(_ context.Context, in *access.GetEventsForHeightRangeRequest) (*access.EventsResponse, error) {
	var events []*access.EventsResponse_Result
	for height := in.StartHeight; height <= in.EndHeight; height++ {
		ee, err := s.index.Events(height, flow.EventType(in.Type))
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

		ee, err := s.index.Events(height, flow.EventType(in.Type))
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

func (s *Server) SendTransaction(ctx context.Context, in *access.SendTransactionRequest) (*access.SendTransactionResponse, error) {
	return nil, errors.New("SendTransaction is not implemented by the Flow DPS API; please use the Flow Access API on a Flow access node directly")
}

func (s *Server) GetLatestProtocolStateSnapshot(ctx context.Context, in *access.GetLatestProtocolStateSnapshotRequest) (*access.ProtocolStateSnapshotResponse, error) {
	return nil, errors.New("GetLatestProtocolSnapshot is not implemented by the Flow DPS API; please use the Flow Access API on a Flow access node directly")
}
