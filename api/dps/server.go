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

	"github.com/go-playground/validator/v10"

	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-dps/models/convert"
	"github.com/onflow/flow-dps/models/dps"
)

// Server is a simple implementation of the generated APIServer interface. It
// uses an index reader interface as the backend to retrieve the desired data.
// This is generally an on-disk interface, but could be a GRPC-based index as
// well, in which case there is a double redirection.
type Server struct {
	index dps.Reader
	codec dps.Codec

	validate *validator.Validate
}

// NewServer creates a new server, using the provided index reader as a backend
// for data retrieval.
func NewServer(index dps.Reader, codec dps.Codec) *Server {

	s := Server{
		index:    index,
		codec:    codec,
		validate: validator.New(),
	}

	return &s
}

// GetFirst implements the `GetFirst` method of the generated GRPC server.
func (s *Server) GetFirst(_ context.Context, _ *GetFirstRequest) (*GetFirstResponse, error) {

	height, err := s.index.First()
	if err != nil {
		return nil, fmt.Errorf("could not get first height: %w", err)
	}

	res := GetFirstResponse{
		Height: height,
	}

	return &res, nil
}

// GetLast implements the `GetLast` method of the generated GRPC server.
func (s *Server) GetLast(_ context.Context, _ *GetLastRequest) (*GetLastResponse, error) {

	height, err := s.index.Last()
	if err != nil {
		return nil, fmt.Errorf("could not get last height: %w", err)
	}

	res := GetLastResponse{
		Height: height,
	}

	return &res, nil
}

// GetHeightForBlock implements the `GetHeightForBlock` method of the generated GRPC
// server.
func (s *Server) GetHeightForBlock(_ context.Context, req *GetHeightForBlockRequest) (*GetHeightForBlockResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	blockID := flow.HashToID(req.BlockID)
	height, err := s.index.HeightForBlock(blockID)
	if err != nil {
		return nil, fmt.Errorf("could not get height for block: %w", err)
	}

	res := GetHeightForBlockResponse{
		BlockID: req.BlockID,
		Height:  height,
	}

	return &res, nil
}

// GetCommit implements the `GetCommit` method of the generated GRPC server.
func (s *Server) GetCommit(_ context.Context, req *GetCommitRequest) (*GetCommitResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	commit, err := s.index.Commit(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get commit: %w", err)
	}

	res := GetCommitResponse{
		Height: req.Height,
		Commit: commit[:],
	}

	return &res, nil
}

// GetHeader implements the `GetHeader` method of the generated GRPC server.
func (s *Server) GetHeader(_ context.Context, req *GetHeaderRequest) (*GetHeaderResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	header, err := s.index.Header(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	data, err := s.codec.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("could not encode header: %w", err)
	}

	res := GetHeaderResponse{
		Height: req.Height,
		Data:   data,
	}

	return &res, nil
}

// GetEvents implements the `GetEvents` method of the generated GRPC server.
func (s *Server) GetEvents(_ context.Context, req *GetEventsRequest) (*GetEventsResponse, error) {

	types := convert.StringsToTypes(req.Types)
	events, err := s.index.Events(req.Height, types...)
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	data, err := s.codec.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("could not encode events: %w", err)
	}

	res := GetEventsResponse{
		Height: req.Height,
		Types:  req.Types,
		Data:   data,
	}

	return &res, nil
}

// GetRegisterValues implements the `GetRegisterValues` method of the
// generated GRPC server.
func (s *Server) GetRegisterValues(_ context.Context, req *GetRegisterValuesRequest) (*GetRegisterValuesResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	paths, err := convert.BytesToPaths(req.Paths)
	if err != nil {
		return nil, fmt.Errorf("could not convert paths: %w", err)
	}

	values, err := s.index.Values(req.Height, paths)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve values: %w", err)
	}

	res := GetRegisterValuesResponse{
		Height: req.Height,
		Paths:  req.Paths,
		Values: convert.ValuesToBytes(values),
	}

	return &res, nil
}

// GetCollection implements the `GetCollection` method of the generated GRPC
// server.
func (s *Server) GetCollection(_ context.Context, req *GetCollectionRequest) (*GetCollectionResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	collID := flow.HashToID(req.CollectionID)
	collection, err := s.index.Collection(collID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve collection: %w", err)
	}

	data, err := s.codec.Marshal(collection)
	if err != nil {
		return nil, fmt.Errorf("could not encode collection: %w", err)
	}

	res := GetCollectionResponse{
		CollectionID: req.CollectionID,
		Data:         data,
	}

	return &res, nil
}

// ListCollectionsForHeight implements the `ListCollectionsForHeight` method of the generated GRPC
// server.
func (s *Server) ListCollectionsForHeight(_ context.Context, req *ListCollectionsForHeightRequest) (*ListCollectionsForHeightResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}
	collIDs, err := s.index.CollectionsByHeight(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not list collections by height: %w", err)
	}

	rawIDs := make([][]byte, 0, len(collIDs))
	for _, collID := range collIDs {
		rawIDs = append(rawIDs, convert.IDToHash(collID))
	}

	res := ListCollectionsForHeightResponse{
		Height:        req.Height,
		CollectionIDs: rawIDs,
	}

	return &res, nil
}

// GetGuarantee implements the `GetGuarantee` method of the generated GRPC
// server.
func (s *Server) GetGuarantee(_ context.Context, req *GetGuaranteeRequest) (*GetGuaranteeResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	collID := flow.HashToID(req.CollectionID)
	guarantee, err := s.index.Guarantee(collID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve guarantee: %w", err)
	}

	data, err := s.codec.Marshal(guarantee)
	if err != nil {
		return nil, fmt.Errorf("could not encode guarantee: %w", err)
	}

	res := GetGuaranteeResponse{
		CollectionID: req.CollectionID,
		Data:         data,
	}

	return &res, nil
}

// GetTransaction implements the `GetTransaction` method of the generated GRPC
// server.
func (s *Server) GetTransaction(_ context.Context, req *GetTransactionRequest) (*GetTransactionResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	txID := flow.HashToID(req.TransactionID)
	transaction, err := s.index.Transaction(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction: %w", err)
	}

	data, err := s.codec.Marshal(transaction)
	if err != nil {
		return nil, fmt.Errorf("could not encode transaction: %w", err)
	}

	res := GetTransactionResponse{
		TransactionID: req.TransactionID,
		Data:          data,
	}

	return &res, nil
}

// GetHeightForTransaction implements the `GetHeightForTransaction` method of the generated GRPC
// server.
func (s *Server) GetHeightForTransaction(_ context.Context, req *GetHeightForTransactionRequest) (*GetHeightForTransactionResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	txID := flow.HashToID(req.TransactionID)
	height, err := s.index.HeightForTransaction(txID)
	if err != nil {
		return nil, fmt.Errorf("could not get height for transaction: %w", err)
	}

	res := GetHeightForTransactionResponse{
		TransactionID: req.TransactionID,
		Height:        height,
	}

	return &res, nil
}

// ListTransactionsForHeight implements the `ListTransactionsForHeight` method of the generated GRPC
// server.
func (s *Server) ListTransactionsForHeight(_ context.Context, req *ListTransactionsForHeightRequest) (*ListTransactionsForHeightResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	txIDs, err := s.index.TransactionsByHeight(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not list transactions by height: %w", err)
	}

	transactionIDs := make([][]byte, 0, len(txIDs))
	for _, txID := range txIDs {
		transactionIDs = append(transactionIDs, convert.IDToHash(txID))
	}

	res := ListTransactionsForHeightResponse{
		Height:         req.Height,
		TransactionIDs: transactionIDs,
	}

	return &res, nil
}

// GetResult implements the `GetResult` method of the generated GRPC
// server.
func (s *Server) GetResult(_ context.Context, req *GetResultRequest) (*GetResultResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	txID := flow.HashToID(req.TransactionID)
	result, err := s.index.Result(txID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve transaction result: %w", err)
	}

	data, err := s.codec.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("could not encode transaction result: %w", err)
	}

	res := GetResultResponse{
		TransactionID: req.TransactionID,
		Data:          data,
	}

	return &res, nil
}

// GetSeal implements the `GetSeal` method of the generated GRPC
// server.
func (s *Server) GetSeal(_ context.Context, req *GetSealRequest) (*GetSealResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	sealID := flow.HashToID(req.SealID)
	seal, err := s.index.Seal(sealID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve seal: %w", err)
	}

	data, err := s.codec.Marshal(seal)
	if err != nil {
		return nil, fmt.Errorf("could not encode seal: %w", err)
	}

	res := GetSealResponse{
		SealID: req.SealID,
		Data:   data,
	}

	return &res, nil
}

// ListSealsForHeight implements the `ListSealsForHeight` method of the generated GRPC
// server.
func (s *Server) ListSealsForHeight(_ context.Context, req *ListSealsForHeightRequest) (*ListSealsForHeightResponse, error) {

	err := s.validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	sealIDs, err := s.index.SealsByHeight(req.Height)
	if err != nil {
		return nil, fmt.Errorf("could not list seals by height: %w", err)
	}

	sIDs := make([][]byte, 0, len(sealIDs))
	for _, sealID := range sealIDs {
		sIDs = append(sIDs, convert.IDToHash(sealID))
	}

	res := ListSealsForHeightResponse{
		Height:  req.Height,
		SealIDs: sIDs,
	}

	return &res, nil
}
