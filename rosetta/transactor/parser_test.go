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

package transactor_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	cjson "github.com/onflow/cadence/encoding/json"
	sdk "github.com/onflow/flow-go-sdk"
	sdkcrypto "github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go/crypto"
	chash "github.com/onflow/flow-go/crypto/hash"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/transactor"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestTransactionParser_BlockID(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()
	index := uint64(84)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			ReferenceBlockID: sdk.HashToID(blockID[:]),
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, tx.ReferenceBlockID.String(), rosBlockID.Hash)

			return index, blockID, nil
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseValidator(validator),
		)

		got, err := p.BlockID()

		require.NoError(t, err)
		assert.Equal(t, tx.ReferenceBlockID.String(), got.Hash)
		assert.Equal(t, index, *got.Index)
	})

	t.Run("handles invalid block data", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			ReferenceBlockID: sdk.HashToID(blockID[:]),
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(identifier.Block) (uint64, flow.Identifier, error) {
			return 0, flow.ZeroID, mocks.GenericError
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseValidator(validator),
		)

		_, err := p.BlockID()

		assert.Error(t, err)
	})
}

func TestTransactionParser_Sequence(t *testing.T) {
	tx := &sdk.Transaction{
		ProposalKey: sdk.ProposalKey{SequenceNumber: 42},
	}

	p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx))

	got := p.Sequence()

	assert.Equal(t, tx.ProposalKey.SequenceNumber, got)
}

func TestTransactionParser_Signers(t *testing.T) {
	header := mocks.GenericHeader
	blockID := header.ID()

	key, err := generateKey()
	require.NoError(t, err)

	// We need to specify a weight of 1000 because otherwise multiple public keys would be required, as
	// the total required weight for an account's signature to be considered valid has to be equal to 1000.
	// For some reason this 1000 magic number is not exposed anywhere that I could find in Flow.
	pubKey := key.PublicKey(1000)

	senderID := mocks.GenericAccountID(0)
	senderAddr := mocks.GenericAddress(0)
	sender := sdk.HexToAddress(senderAddr.Hex())
	receiver := sdk.HexToAddress(mocks.GenericAddress(1).Hex())
	signature := sdk.TransactionSignature{
		Address:     sender,
		SignerIndex: 64,
		KeyIndex:    128,
	}

	invoker := mocks.BaselineInvoker(t)
	invoker.KeyFunc = func(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {
		return &pubKey, nil
	}

	tx := &sdk.Transaction{
		ReferenceBlockID:   sdk.HashToID(blockID[:]),
		Authorizers:        []sdk.Address{sender},
		EnvelopeSignatures: []sdk.TransactionSignature{signature},
	}

	signer := sdkcrypto.NewInMemorySigner(key.PrivateKey, key.HashAlgo)
	message := tx.EnvelopeMessage()
	message = append(sdk.TransactionDomainTag[:], message...)

	sig, err := signer.Sign(message)
	require.NoError(t, err)

	tx.EnvelopeSignatures[0].Signature = sig

	t.Run("nominal case with unsigned transaction", func(t *testing.T) {
		t.Parallel()

		p := transactor.BaselineTransactionParser(t,
			transactor.ParseTransaction(sdk.NewTransaction()),
			transactor.ParseInvoker(invoker),
		)

		got, err := p.Signers()

		require.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("nominal case with signed transaction", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.KeyFunc = func(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {
			assert.Equal(t, header.Height, height)
			assert.Equal(t, senderAddr, address)
			assert.Zero(t, index)

			return &pubKey, nil
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx), transactor.ParseInvoker(invoker))

		got, err := p.Signers()

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, senderID, got[0])
	})

	t.Run("handles case where transaction contains a payload signature (which it should not)", func(t *testing.T) {
		t.Parallel()

		// Copy the valid transaction and only add a payload signature to it to trigger a failure.

		tx := &sdk.Transaction{
			ReferenceBlockID:   sdk.HashToID(blockID[:]),
			Authorizers:        []sdk.Address{sender},
			EnvelopeSignatures: []sdk.TransactionSignature{signature},
			PayloadSignatures:  []sdk.TransactionSignature{signature},
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx), transactor.ParseInvoker(invoker))

		_, err := p.Signers()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles case where there are multiple envelope signatures", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			ReferenceBlockID:   sdk.HashToID(blockID[:]),
			Authorizers:        []sdk.Address{sender},
			EnvelopeSignatures: []sdk.TransactionSignature{signature, signature, signature},
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx), transactor.ParseInvoker(invoker))

		_, err := p.Signers()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles case where transaction signer is not the sender", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			ReferenceBlockID:   sdk.HashToID(blockID[:]),
			Authorizers:        []sdk.Address{receiver},
			EnvelopeSignatures: []sdk.TransactionSignature{signature},
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx), transactor.ParseInvoker(invoker))

		_, err := p.Signers()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles case where transaction has invalid block ID", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(identifier.Block) (uint64, flow.Identifier, error) {
			return 0, flow.ZeroID, mocks.GenericError
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseValidator(validator),
			transactor.ParseInvoker(invoker),
		)

		_, err := p.Signers()

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on Key", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.KeyFunc = func(uint64, flow.Address, int) (*flow.AccountPublicKey, error) {
			return nil, mocks.GenericError
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseInvoker(invoker),
		)

		_, err := p.Signers()

		assert.Error(t, err)
	})

	t.Run("handles signature and key mismatch", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.KeyFunc = func(uint64, flow.Address, int) (*flow.AccountPublicKey, error) {
			// This is not the signature that was used to sign the data in the envelope, so
			// the verification should fail.
			mockSignature := mocks.GenericAccount.Keys[0]

			return &mockSignature, nil
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx), transactor.ParseInvoker(invoker))

		_, err := p.Signers()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles invalid sender account", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(identifier.Account) (flow.Address, error) {
			return flow.EmptyAddress, mocks.GenericError
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseInvoker(invoker),
			transactor.ParseValidator(validator),
		)

		_, err := p.Signers()

		assert.Error(t, err)
	})
}

func TestTransactionParser_Operations(t *testing.T) {
	sender := sdk.HexToAddress(mocks.GenericAddress(0).Hex())
	receiverAddr := mocks.GenericAddress(1)
	receiver := sdk.HexToAddress(receiverAddr.Hex())

	amount := mocks.GenericAmount(0)
	amountData, err := cjson.Encode(amount)
	require.NoError(t, err)

	cadenceAddr := cadence.BytesToAddress(receiverAddr.Bytes())
	addressData, err := cjson.Encode(cadenceAddr)
	require.NoError(t, err)

	tx := &sdk.Transaction{
		Payer:       sender,
		ProposalKey: sdk.ProposalKey{Address: sender},
		Authorizers: []sdk.Address{sender},
		Script:      mocks.GenericBytes,
		Arguments:   [][]byte{amountData, addressData},
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx))

		got, err := p.Operations()

		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("handles invalid number of authorizers", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Authorizers: []sdk.Address{sender, receiver, sender},
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx))

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidAuthorizers{})
	})

	t.Run("handles invalid authorizer account", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Authorizers: []sdk.Address{sender},
		}

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(identifier.Account) (flow.Address, error) {
			return flow.EmptyAddress, mocks.GenericError
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseValidator(validator),
		)

		_, err := p.Operations()

		assert.Error(t, err)
	})

	t.Run("handles mismatch between authorizer and payer", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       receiver,
			Authorizers: []sdk.Address{sender},
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx))

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayer{})
	})

	t.Run("handles mismatch between proposal key address and payer", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: receiver},
			Authorizers: []sdk.Address{sender},
		}

		p := transactor.BaselineTransactionParser(t, transactor.ParseTransaction(tx))

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidProposer{})
	})

	t.Run("handles transfer token generation failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TransferTokensFunc = func(string) ([]byte, error) {
			return nil, mocks.GenericError
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseGenerator(generator),
		)

		_, err := p.Operations()

		assert.Error(t, err)
	})

	t.Run("handles transfer token script mismatch", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TransferTokensFunc = func(string) ([]byte, error) {
			return []byte{}, nil
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
			transactor.ParseGenerator(generator),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidScript{})
	})

	t.Run("handles invalid number of arguments (0)", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: sender},
			Authorizers: []sdk.Address{sender},
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{}, // No argument.
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidArguments{})
	})

	t.Run("handles invalid number of arguments (<)", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: sender},
			Authorizers: []sdk.Address{sender},
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{amountData}, // Only one argument.
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidArguments{})
	})

	t.Run("handles invalid number of arguments (>)", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: sender},
			Authorizers: []sdk.Address{sender},
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{amountData, addressData, amountData}, // Three arguments.
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidArguments{})
	})

	t.Run("handles invalid amount argument (not a uint)", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: sender},
			Authorizers: []sdk.Address{sender},
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{addressData, addressData}, // First argument is not an uint.
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidAmount{})
	})

	t.Run("handles invalid amount argument (not json-encoded)", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: sender},
			Authorizers: []sdk.Address{sender},
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{mocks.GenericBytes, addressData}, // First argument is not json-encoded.
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidAmount{})
	})

	t.Run("handles invalid address argument (not json-encoded)", func(t *testing.T) {
		t.Parallel()

		tx := &sdk.Transaction{
			Payer:       sender,
			ProposalKey: sdk.ProposalKey{Address: sender},
			Authorizers: []sdk.Address{sender},
			Script:      mocks.GenericBytes,
			Arguments:   [][]byte{amountData, mocks.GenericBytes}, // Second argument is not json-encoded.
		}

		p := transactor.BaselineTransactionParser(
			t,
			transactor.ParseTransaction(tx),
		)

		_, err := p.Operations()

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidReceiver{})
	})
}

func generateKey() (*flow.AccountPrivateKey, error) {
	seed := make([]byte, crypto.KeyGenSeedMaxLenECDSA)

	_, err := rand.Read(seed)
	if err != nil {
		return nil, err
	}

	signAlgo := crypto.ECDSAP256
	hashAlgo := chash.SHA3_256

	key, err := crypto.GeneratePrivateKey(signAlgo, seed)
	if err != nil {
		return nil, err
	}

	return &flow.AccountPrivateKey{
		PrivateKey: key,
		SignAlgo:   key.Algorithm(),
		HashAlgo:   hashAlgo,
	}, nil
}
