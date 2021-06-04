# Rosetta API

The Rosetta API needs its own documentation because of the amount of components it has that interact with each other.
The main reason for its complexity is that it needs to interact with the Flow Virtual Machine (FVM) and to translate between the Flow and Rosetta application domains.

## Invoker

This component, given a Cadence script, can execute it at any given height and return the value produced by the script.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/rosetta/invoker)

## Retriever

The retriever uses the other components to retrieve account balances, blocks and transactions.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/rosetta/retriever)

## Scripts

The script package produces Cadence scripts with the correct imports and storage paths, depending on the configured Flow chain ID.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/rosetta/scripts)

## Validator

The Validator component validates whether the given Rosetta identifiers are valid.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/rosetta/validator)

