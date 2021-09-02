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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	sdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/object"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/transactor"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestTransactor_DeriveIntent(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		want := mocks.GenericOperations(2)
		got, err := tr.DeriveIntent(want)

		require.NoError(t, err)
		assert.Equal(t, want[1].Amount.Value, fmt.Sprint(uint64(got.Amount)))
		assert.Equal(t, want[1].AccountID.Address, got.To.String())
		assert.Equal(t, want[0].AccountID.Address, got.From.String())
		assert.Equal(t, want[0].AccountID.Address, got.Payer.String())
		assert.Equal(t, want[0].AccountID.Address, got.Proposer.String())
	})

	t.Run("handles invalid currency", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.CurrencyFunc = func(identifier.Currency) (string, uint, error) {
			return "", 0, mocks.GenericError
		}

		tr := transactor.BaselineTransactor(t, transactor.WithValidator(validator))

		_, err := tr.DeriveIntent(mocks.GenericOperations(2))

		assert.Error(t, err)
	})

	t.Run("handles invalid account", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(account identifier.Account) (flow.Address, error) {
			return flow.EmptyAddress, mocks.GenericError
		}

		tr := transactor.BaselineTransactor(t, transactor.WithValidator(validator))

		_, err := tr.DeriveIntent(mocks.GenericOperations(2))

		assert.Error(t, err)
	})

	t.Run("handles invalid number of operations", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		op := mocks.GenericOperations(3)

		_, err := tr.DeriveIntent(op)

		assert.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidOperations{})
	})

	t.Run("handles operations with unparsable amounts", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		op := mocks.GenericOperations(2)
		op[0].Amount.Value = "42"
		op[1].Amount.Value = "84"

		_, err := tr.DeriveIntent(op)

		assert.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidIntent{})
	})

	t.Run("handles operations with non-matching amounts", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		op := mocks.GenericOperations(2)
		op[0].Amount.Value = "42"
		op[1].Amount.Value = "84"

		_, err := tr.DeriveIntent(op)

		assert.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidIntent{})
	})

	t.Run("handles irrelevant currencies", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.CurrencyFunc = func(identifier.Currency) (string, uint, error) {
			return "IRRELEVANT_CURRENCY", 0, nil
		}

		tr := transactor.BaselineTransactor(t, transactor.WithValidator(validator))

		_, err := tr.DeriveIntent(mocks.GenericOperations(2))

		assert.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidIntent{})
	})

	t.Run("handles non-transfer operations", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		op := mocks.GenericOperations(2)
		op[0].Type = "irrelevant_type"

		_, err := tr.DeriveIntent(op)

		assert.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidIntent{})
	})
}

func TestTransactor_CompileTransaction(t *testing.T) {
	rosBlockID := mocks.GenericRosBlockID
	sequence := uint64(42)

	sender := mocks.GenericAddress(0)
	receiver := mocks.GenericAddress(1)
	amount, err := cadence.NewUFix64("100.00000000")
	require.NoError(t, err)

	intent := &transactor.Intent{
		From:     sender,
		To:       receiver,
		Amount:   amount,
		Payer:    sender,
		Proposer: sender,
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		wantCompiled := `eyJTY3JpcHQiOiJkR1Z6ZEE9PSIsIkFyZ3VtZW50cyI6WyJleUowZVhCbElqb2lWVVpwZURZMElpd2lkbUZzZFdVaU9pSXhNREF1TURBd01EQXdNREFpZlFvPSIsImV5SjBlWEJsSWpvaVFXUmtjbVZ6Y3lJc0luWmhiSFZsSWpvaU1IaGpNamd5WlRJeVl6bGlNbVExTTJObUluMEsiXSwiUmVmZXJlbmNlQmxvY2tJRCI6WzcsNDAsMjgsMTUyLDIyOSwxNCwxMzMsNzgsOCwxOCwyNTIsMTI1LDExNywyMjAsMjIwLDY2LDE3NCwxNjIsMTUxLDcxLDEyNSw4OSw5MCw0MSwyMDIsMTE1LDY5LDIxOCwyMDIsMzYsMTQzLDU0XSwiR2FzTGltaXQiOjk5OTksIlByb3Bvc2FsS2V5Ijp7IkFkZHJlc3MiOiJlNmU0NjMyYWUwMTMwOWMwIiwiS2V5SW5kZXgiOjAsIlNlcXVlbmNlTnVtYmVyIjo0Mn0sIlBheWVyIjoiZTZlNDYzMmFlMDEzMDljMCIsIkF1dGhvcml6ZXJzIjpbImU2ZTQ2MzJhZTAxMzA5YzAiXSwiUGF5bG9hZFNpZ25hdHVyZXMiOm51bGwsIkVudmVsb3BlU2lnbmF0dXJlcyI6bnVsbH0=`

		generator := mocks.BaselineGenerator(t)
		generator.TransferTokensFunc = func(symbol string) ([]byte, error) {
			assert.Equal(t, dps.FlowSymbol, symbol)

			return mocks.GenericBytes, nil
		}

		tr := transactor.BaselineTransactor(t, transactor.WithGenerator(generator))

		got, err := tr.CompileTransaction(rosBlockID, intent, sequence)

		require.NoError(t, err)
		assert.Equal(t, wantCompiled, got)
	})

	t.Run("handles generator failure on TransferTokens", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TransferTokensFunc = func(string) ([]byte, error) {
			return nil, mocks.GenericError
		}

		tr := transactor.BaselineTransactor(t, transactor.WithGenerator(generator))

		_, err := tr.CompileTransaction(rosBlockID, intent, sequence)

		assert.Error(t, err)
	})
}

func TestTransactor_HashPayload(t *testing.T) {
	header := mocks.GenericHeader
	rosBlockID := mocks.GenericRosBlockID
	signer := mocks.GenericAccountID(0)
	signerAddr := mocks.GenericAddress(0)
	tx := &sdk.Transaction{
		ProposalKey: sdk.ProposalKey{SequenceNumber: 42},
	}

	key, err := generateKey()
	require.NoError(t, err)

	// We need to specify a weight of 1000 because otherwise multiple public keys would be required, as
	// the total required weight for an account's signature to be considered valid has to be equal to 1000.
	// For some reason this 1000 magic number is not exposed anywhere that I could find in Flow.
	pubKey := key.PublicKey(1000)

	data, err := json.Marshal(tx)
	require.NoError(t, err)

	payload := base64.StdEncoding.EncodeToString(data)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(gotBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, rosBlockID, gotBlockID)

			return header.Height, header.ID(), nil
		}
		validator.AccountFunc = func(rosAccountID identifier.Account) (flow.Address, error) {
			assert.Equal(t, signer, rosAccountID)

			return signerAddr, nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.KeyFunc = func(height uint64, address flow.Address, index int) (*flow.AccountPublicKey, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			assert.Equal(t, signerAddr, address)
			assert.Zero(t, index)

			return &pubKey, nil
		}

		tr := transactor.BaselineTransactor(
			t,
			transactor.WithValidator(validator),
			transactor.WithInvoker(invoker),
		)

		algorithm, hash, err := tr.HashPayload(rosBlockID, payload, signer)

		require.NoError(t, err)
		assert.Equal(t, "ecdsa", algorithm)
		assert.Equal(t, "3395952d355d9a5e9b5ba6f46f155cca9ec7615deef5ce1146926cb4abbf5cbb", hash)
	})

	t.Run("handles non-base64-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		_, _, err = tr.HashPayload(rosBlockID, string(data), signer)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles non-json-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload := base64.StdEncoding.EncodeToString(mocks.GenericBytes)

		_, _, err = tr.HashPayload(rosBlockID, invalidPayload, signer)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(identifier.Block) (uint64, flow.Identifier, error) {
			return 0, flow.ZeroID, mocks.GenericError
		}

		tr := transactor.BaselineTransactor(
			t,
			transactor.WithValidator(validator),
		)

		_, _, err := tr.HashPayload(rosBlockID, payload, signer)

		assert.Error(t, err)
	})

	t.Run("handles invalid account", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(identifier.Account) (flow.Address, error) {
			return flow.EmptyAddress, mocks.GenericError
		}

		tr := transactor.BaselineTransactor(
			t,
			transactor.WithValidator(validator),
		)

		_, _, err := tr.HashPayload(rosBlockID, payload, signer)

		assert.Error(t, err)
	})

	t.Run("handles invoker failure on Key", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.KeyFunc = func(uint64, flow.Address, int) (*flow.AccountPublicKey, error) {
			return nil, mocks.GenericError
		}

		tr := transactor.BaselineTransactor(
			t,
			transactor.WithInvoker(invoker),
		)

		_, _, err := tr.HashPayload(rosBlockID, payload, signer)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidKey{})
	})
}

func TestTransactor_Parse(t *testing.T) {
	tx := &sdk.Transaction{
		ProposalKey: sdk.ProposalKey{SequenceNumber: 42},
	}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		encodedData := base64.StdEncoding.EncodeToString(data)

		tr := transactor.BaselineTransactor(t)

		got, err := tr.Parse(encodedData)

		require.NoError(t, err)
		assert.Equal(t, tx.ProposalKey.SequenceNumber, got.Sequence())
	})

	t.Run("handles missing base64 encoding", func(t *testing.T) {
		t.Parallel()

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		tr := transactor.BaselineTransactor(t)

		_, err = tr.Parse(string(data))

		assert.Error(t, err)
	})

	t.Run("handles non-JSON-encoded payloads", func(t *testing.T) {
		t.Parallel()

		payload := mocks.GenericBytes

		tr := transactor.BaselineTransactor(t)

		_, err := tr.Parse(string(payload))

		assert.Error(t, err)
	})
}

func TestTransactor_AttachSignatures(t *testing.T) {
	senderID := mocks.GenericAccountID(0)
	sender := sdk.HexToAddress(mocks.GenericAddress(0).Hex())
	receiverID := mocks.GenericAccountID(1)
	receiver := sdk.HexToAddress(mocks.GenericAddress(1).Hex())
	tx := &sdk.Transaction{
		Authorizers: []sdk.Address{
			sender,
		},
		Payer: sender,
	}

	data, err := json.Marshal(tx)
	require.NoError(t, err)

	payload := base64.StdEncoding.EncodeToString(data)

	key, err := generateKey()
	require.NoError(t, err)

	// We need to specify a weight of 1000 because otherwise multiple public keys would be required, as
	// the total required weight for an account's signature to be considered valid has to be equal to 1000.
	// For some reason this 1000 magic number is not exposed anywhere that I could find in Flow.
	pubKey := key.PublicKey(1000)
	hexBytes := strings.TrimPrefix(pubKey.PublicKey.String(), "0x")
	senderSignature := object.Signature{
		SigningPayload: object.SigningPayload{
			AccountID:     senderID,
			HexBytes:      hexBytes,
			SignatureType: "ecdsa",
		},
		SignatureType: "ecdsa",
		HexBytes:      hexBytes,
		PublicKey: object.PublicKey{
			HexBytes: hexBytes,
		},
	}
	receiverSignature := object.Signature{
		SigningPayload: object.SigningPayload{
			AccountID:     receiverID,
			HexBytes:      hexBytes,
			SignatureType: "ecdsa",
		},
		SignatureType: "ecdsa",
		HexBytes:      hexBytes,
		PublicKey: object.PublicKey{
			HexBytes: hexBytes,
		},
	}
	signatures := []object.Signature{senderSignature}

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		got, err := tr.AttachSignatures(payload, signatures)

		require.NoError(t, err)
		assert.NotEmpty(t, got)
	})

	t.Run("handles non-base64-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload, err := json.Marshal(tx)
		require.NoError(t, err)

		_, err = tr.AttachSignatures(string(invalidPayload), signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles non-json-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload := base64.StdEncoding.EncodeToString(mocks.GenericBytes)

		_, err = tr.AttachSignatures(invalidPayload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles invalid number of authorizers (>)", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		tx := &sdk.Transaction{
			Authorizers: []sdk.Address{
				sender,
				sender,
			},
			Payer: sender,
		}

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		payload := base64.StdEncoding.EncodeToString(data)

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidAuthorizers{})
	})

	t.Run("handles invalid number of authorizers (0)", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		tx := &sdk.Transaction{
			Authorizers: []sdk.Address{},
			Payer:       sender,
		}

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		payload := base64.StdEncoding.EncodeToString(data)

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidAuthorizers{})
	})

	t.Run("handles mismatch between length of signatures and authorizers", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		signatures := []object.Signature{senderSignature, senderSignature, senderSignature} // 3 signatures but only 1 authorizer.

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignatures{})
	})

	t.Run("handles mismatch between payer and sender", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		tx := &sdk.Transaction{
			Authorizers: []sdk.Address{sender},
			Payer:       receiver,
		}

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		payload := base64.StdEncoding.EncodeToString(data)

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayer{})
	})

	t.Run("handles unexpected envelope signatures", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		tx := &sdk.Transaction{
			Authorizers:        []sdk.Address{sender},
			Payer:              sender,
			EnvelopeSignatures: []sdk.TransactionSignature{{}}, // 1 empty senderSignature just to trigger the failure.
		}

		data, err := json.Marshal(tx)
		require.NoError(t, err)

		payload := base64.StdEncoding.EncodeToString(data)

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles mismatch between signer and sender", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		signatures := []object.Signature{receiverSignature} // Signed by receiver instead of sender.

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles invalid signature algorithm", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		signatures := []object.Signature{
			{
				SigningPayload: object.SigningPayload{
					AccountID:     senderID,
					HexBytes:      hexBytes,
					SignatureType: "invalid_type",
				},
				SignatureType: "invalid_type",
				HexBytes:      hexBytes,
				PublicKey: object.PublicKey{
					HexBytes: hexBytes,
				},
			},
		}

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})

	t.Run("handles invalid signature bytes", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		signatures := []object.Signature{
			{
				SigningPayload: object.SigningPayload{
					AccountID:     senderID,
					HexBytes:      string(mocks.GenericBytes),
					SignatureType: "ecdsa",
				},
				SignatureType: "ecdsa",
				HexBytes:      string(mocks.GenericBytes),
				PublicKey: object.PublicKey{
					HexBytes: string(mocks.GenericBytes),
				},
			},
		}

		_, err = tr.AttachSignatures(payload, signatures)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidSignature{})
	})
}

func TestTransactor_TransactionIdentifier(t *testing.T) {
	tx := &sdk.Transaction{}

	data, err := json.Marshal(tx)
	require.NoError(t, err)

	payload := base64.StdEncoding.EncodeToString(data)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		got, err := tr.TransactionIdentifier(payload)

		require.NoError(t, err)
		assert.Equal(t, tx.ID().Hex(), got.Hash)
	})

	t.Run("handles non-base64-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload, err := json.Marshal(tx)
		require.NoError(t, err)

		_, err = tr.TransactionIdentifier(string(invalidPayload))

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles non-json-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload := base64.StdEncoding.EncodeToString(mocks.GenericBytes)

		_, err = tr.TransactionIdentifier(invalidPayload)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})
}

func TestTransactor_SubmitTransaction(t *testing.T) {
	tx := &sdk.Transaction{}

	data, err := json.Marshal(tx)
	require.NoError(t, err)

	payload := base64.StdEncoding.EncodeToString(data)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		submitter := mocks.BaselineSubmitter(t)
		submitter.TransactionFunc = func(gotTx *sdk.Transaction) error {
			assert.Equal(t, tx, gotTx)

			return nil
		}

		tr := transactor.BaselineTransactor(t, transactor.WithSubmitter(submitter))

		got, err := tr.SubmitTransaction(payload)

		require.NoError(t, err)
		assert.Equal(t, tx.ID().Hex(), got.Hash)
	})

	t.Run("handles non-base64-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload, err := json.Marshal(tx)
		require.NoError(t, err)

		_, err = tr.SubmitTransaction(string(invalidPayload))

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles non-json-encoded transaction payload", func(t *testing.T) {
		t.Parallel()

		tr := transactor.BaselineTransactor(t)

		invalidPayload := base64.StdEncoding.EncodeToString(mocks.GenericBytes)

		_, err = tr.SubmitTransaction(invalidPayload)

		require.Error(t, err)
		assert.ErrorAs(t, err, &failure.InvalidPayload{})
	})

	t.Run("handles submitter failure", func(t *testing.T) {
		t.Parallel()

		submitter := mocks.BaselineSubmitter(t)
		submitter.TransactionFunc = func(*sdk.Transaction) error {
			return mocks.GenericError
		}

		tr := transactor.BaselineTransactor(t, transactor.WithSubmitter(submitter))

		_, err := tr.SubmitTransaction(payload)

		assert.Error(t, err)
	})
}
