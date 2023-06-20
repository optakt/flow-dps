// Utility to validate Access API and Archive-Access API return values

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type APIValidator struct {
	ctx           context.Context
	archiveClient access.AccessAPIClient
	accessClient  access.AccessAPIClient
	script        []byte
	arguments     [][]byte
	blockID       []byte
	blockHeight   uint64
	accountAddr   []byte
}

func NewAPIValidator(accessAddr string, archiveAddr string, ctx context.Context) (*APIValidator, error) {
	accessClient := getAPIClient(accessAddr)
	archiveClient := getAPIClient(archiveAddr)
	accountAddr := flow.HexToAddress("e467b9dd11fa00df").Bytes()
	recentBlock, err := accessClient.GetBlockByHeight(ctx, &access.GetBlockByHeightRequest{Height: 106242795}) // this might be flakey because a sealed block to access node might not be sealed yet to archive node
	// recentBlock, err := accessClient.GetBlockByHeight(ctx, &access.GetBlockByHeightRequest{Height: 52853749}) // specify with a recent sealed block
	// allow for archive node to sync block
	if err != nil {
		return nil, fmt.Errorf("unable to get latest block from AN")
	}
	blockID := recentBlock.GetBlock().GetId()
	blockHeight := recentBlock.GetBlock().Height
	scriptPath := "get_token_balance.cdc"
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read cadence script to initilaize: %w", err)
	}
	scriptArgs := make([][]byte, 0)
	return &APIValidator{
		ctx:           ctx,
		accountAddr:   accountAddr,
		script:        script,
		blockID:       blockID,
		arguments:     scriptArgs,
		blockHeight:   blockHeight,
		accessClient:  accessClient,
		archiveClient: archiveClient,
	}, nil
}

func getAPIClient(addr string) access.AccessAPIClient {
	// connect to Archive-Access instance
	MaxGRPCMessageSize := 1024 * 1024 * 20 // 20MB
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize)))
	if err != nil {
		panic(fmt.Sprintf("unable to create connection to node: %s", addr))
	}
	return access.NewAccessAPIClient(conn)
}

func (a *APIValidator) CheckAPIResults(ctx context.Context) error {
	log.Info().Msgf("starting comparison for block %v (%x)", a.blockHeight, a.blockID)
	// ExecuteScriptAtBlockID
	err := a.checkExecuteScriptAtBlockID(ctx)
	if err != nil {
		return fmt.Errorf("unsuccessful ExecuteScriptAtBlockID comparison: %w", err)
	}
	log.Info().Msg("checkExecuteScriptAtBlockID successful")

	// ExecuteScriptAtBlockHeight
	err = a.checkExecuteScriptAtBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("unsuccessful ExecuteScriptAtBlockHeight comparison: %w", err)
	}
	log.Info().Msg("checkExecuteScriptAtBlockHeight successful")

	// GetAccountAtBlockHeight
	err = a.checkGetAccountAtBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("unsuccessful checkGetAccountAtBlockHeight comparison: %w", err)
	}
	log.Info().Msg("checkGetAccountAtBlockHeight successful")
	return nil
}

func (a *APIValidator) checkExecuteScriptAtBlockID(ctx context.Context) error {
	var errs *multierror.Error

	req := &access.ExecuteScriptAtBlockIDRequest{
		BlockId:   a.blockID,
		Script:    a.script,
		Arguments: a.arguments,
	}
	accessRes, accessErr := a.accessClient.ExecuteScriptAtBlockID(ctx, req)
	if accessErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to get ExecuteScriptAtBlockID from access node: %w", accessErr))
	}
	log.Debug().Msg(fmt.Sprintf("received ExecuteScriptAtBlockID response from AN: %s", accessRes.String()))

	archiveRes, archiveErr := a.archiveClient.ExecuteScriptAtBlockID(ctx, req)
	if archiveErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to get ExecuteScriptAtBlockID from archive node: %w", archiveErr))
	}
	log.Debug().Msg(fmt.Sprintf("received ExecuteScriptAtBlockID response from Archive: %s", archiveRes.String()))

	if errs != nil {
		return errs.ErrorOrNil()
	}

	if accessRes.String() != archiveRes.String() {
		return fmt.Errorf("unequal results! for ExecuteScriptAtBlockID")
	}

	return nil
}

func (a *APIValidator) checkExecuteScriptAtBlockHeight(ctx context.Context) error {
	var errs *multierror.Error

	req := &access.ExecuteScriptAtBlockHeightRequest{
		BlockHeight: a.blockHeight,
		Script:      a.script,
		Arguments:   a.arguments,
	}

	accessRes, accessErr := a.accessClient.ExecuteScriptAtBlockHeight(ctx, req)
	if accessErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to get ExecuteScriptAtBlockHeight from access node: %w", accessErr))
	}
	log.Debug().Msg(fmt.Sprintf("received ExecuteScriptAtBlockHeight response from AN: %s", accessRes.String()))

	archiveRes, archiveErr := a.archiveClient.ExecuteScriptAtBlockHeight(ctx, req)
	if archiveErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to get ExecuteScriptAtBlockHeight from archive node: %w", archiveErr))
	}
	log.Debug().Msg(fmt.Sprintf("received ExecuteScriptAtBlockHeight response from Archive: %s", archiveRes.String()))

	if errs != nil {
		return errs.ErrorOrNil()
	}

	if accessRes.String() != archiveRes.String() {
		return fmt.Errorf("unequal results! for ExecuteScriptAtBlockHeight")
	}

	return nil

}

func (a *APIValidator) checkGetAccountAtBlockHeight(ctx context.Context) error {
	req := &access.GetAccountAtBlockHeightRequest{
		Address:     a.accountAddr,
		BlockHeight: a.blockHeight,
	}
	accessRes, err := a.accessClient.GetAccountAtBlockHeight(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get GetAccountAtBlockHeight from access node: %w", err)
	}
	log.Debug().Msg(fmt.Sprintf("received GetAccountAtBlockHeight response from AN: %s", accessRes.String()))
	archiveRes, err := a.archiveClient.GetAccountAtBlockHeight(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get GetAccountAtBlockHeight from access node: %w", err)
	}
	log.Debug().Msg(fmt.Sprintf("received GetAccountAtBlockHeight response from Archive: %s", archiveRes.String()))
	if accessRes.String() != archiveRes.String() {
		return fmt.Errorf("unequal results! GetAccountAtBlockHeight from access node: %w", err)
	}
	return nil
}

func main() {
	// connect to Archive-Access instance
	ctx := context.TODO()
	accessAddr := "access-001.devnet46.nodes.onflow.org:9000"
	archiveAddr := "access-003.devnet46.nodes.onflow.org:9000"
	// archiveAddr := "dps-001.mainnet-staging1.nodes.onflow.org:9000" // badger based archive node
	// archiveAddr := "dps-001.mainnet22.nodes.onflow.org:9000" // existing dps with the trie
	// connect to Access instance
	apiValidator, err := NewAPIValidator(accessAddr, archiveAddr, ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize validator")
		return
	}
	// compare
	err = apiValidator.CheckAPIResults(ctx)
	if err != nil {
		print(err.Error())
		log.Info().Err(fmt.Errorf("error while comparing API responses: %w", err))
		return
	}
	log.Info().Msg("comparison successful, Archive and AN results match")
}
