# Architecture

This document describes the internal components that the Flow Data Provisioning Service is constituted of, as well as the API it exposes.

**Table of Contents**

1. [Chain](#chain)
    1. [ProtocolState](#ProtocolState-chain)
2. [Feeder](#feeder)
    1. [LedgerWAL](#LedgerWAL-feeder)
3. [Mapper](#mapper)
4. [Store](#store)
    1. [Database Schema](#database-schema)
        1. [Block-To-Height Index](#block-to-height-index)
        2. [Commit-To-Height Index](#commit-to-height-index)
        3. [Height-To-Commit Index](#height-to-commit-index)
        4. [Header Index](#header-index)
        5. [Path Deltas Index](#path-deltas-index)
        6. [Events Index](#events-index)
5. [API](#api)
    1. [Rosetta API](#rosetta-api)
        1. [Contracts](#contracts)
        2. [Scripts](#scripts)
        3. [Invoker](#invoker)
        4. [Validator](#validator)
        5. [Retriever](#retriever)

## Chain

The Chain component is responsible for reconstructing a view of the sequence of blocks, along with their metadata.
It allows the consumer to step from the root block to the last sealed block, while presenting height, block identifier and state commitment for each step.
It is used by the [Mapper](#mapper) to map blocks to the deltas that are collected by the [Feeder](#feeder) component.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/chain)

### ProtocolState Chain

The [Filesystem Chain](https://pkg.go.dev/github.com/awfm9/flow-dps/chain#ProtocolState) uses the execution node's on-disk key-value store to reconstruct the block sequence.

## Feeder

The Feeder component is responsible for streaming in-order trie updates.
It outputs deltas which are used by the [Mapper](#mapper) component to map the state trie updates to the block information that the [Chain](#chain) component provides.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/feeder)

### LedgerWAL Feeder

The [LedgerWAL Feeder](https://pkg.go.dev/github.com/awfm9/flow-dps/feeder#LedgerWAL) reads trie updates from the LedgerWAL directly.

## Mapper

The mapper component is at the core of the DPS. It is responsible for mapping incoming state trie updates to blocks.
In order to do that, it depends on the [Feeder](#feeder) and [Chain](#chain) components to get state trie updates and block information, as well as on the [Store](#store) component for indexing.
Generally, trie updates come in by chunk, so each block maps from zero to multiple trie updates.
Once a block is mapped to its respective trie updates, the mapper forwards the information to the indexer.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/mapper)

## Store

The Store component is responsible for receiving a set of trie updates for each block and creating the necessary main indexes and auxiliary in the on-disk database.
These indexes allow efficient retrieval of the state at arbitrary block heights in the state history.
It also provides random access to the execution state by providing smart access to these indexes.
It combines writing and retrieving of indexes, so that an efficient caching strategy is possible.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/indexer)

### Database Schema

The DPS uses [BadgerDB](https://github.com/dgraph-io/badger) to store datasets of state changes and block information to build all the indexes required for random protocol and execution state access.
It does not re-use any of the protocol state database, but instead re-indexes everything, so that all databases used to bootstrap can be discarded subsequently.

#### Block-To-Height Index

In this index, keys map the block ID to the block height.

| **Length** (bytes) | `1`               | `8`        |
|:-------------------|:------------------|:-----------|
| **Type**           | byte              | hex hash   |
| **Description**    | Index type prefix | Block ID   |
| **Example Value**  | `2`               | `1fd5532a` |

The value stored at that key is the **Height** of the referenced block.

##### Commit-To-Height Index

In this index, keys map the state commitment hash to the block height.

| **Length** (bytes) | `1`               | `8`        |
|:-------------------|:------------------|:-----------|
| **Type**           | byte              | hex hash   |
| **Description**    | Index type prefix | Commit     |
| **Example Value**  | `3`               | `3f5d8120` |

The value stored at that key is the **Height** of the referenced state commitment's block.

##### Height-To-Commit Index

In this index, keys map the block height to the state commitment hash.

| **Length** (bytes) | `1`               | `8`          |
|:-------------------|:------------------|:-------------|
| **Type**           | byte              | uint64       |
| **Description**    | Index type prefix | Block Height |
| **Example Value**  | `4`               | `425`        |

The value stored at that key is the **state commitment hash** of the referenced block height.

##### Header Index

In order to provide an efficient implementation of the Rosetta API, this index maps block heights to block headers.
The header contains the metadata for a block as well as a hash representing the combined payload of the entire block.

| **Length (bytes)** | `1`               | `8`          |
|:-------------------|:------------------|:-------------|
| **Type**           | uint              | uint64       |
| **Description**    | Index type prefix | Block Height |
| **Example Value**  | `5`               | `425`        |

The value stored at that key is the **Height** of the referenced state commitment's block.

##### Path Deltas Index

This index maps a block ID to all the paths that are changed within its state updates.

| **Length (bytes)** | `1`               | `pathfinder.PathByteSize` | `8`          |
|:-------------------|:------------------|:--------------------------|:-------------|
| **Type**           | uint              |          string           | uint64       |
| **Description**    | Index type prefix |       Register path       | Block Height |
| **Example Value**  | `6`               |      `/0//1//2/uuid`      | `425`        |

The value stored at that key is **the compressed payload of the change at the given path**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

##### Events Index

The events index indexes events grouped by block height and transaction type.
The block height is first in the index so that we can look through all events at a given height regardless of type using a key prefix.

| **Length (bytes)** | `1`               | `8`          | `64`                        |
|:-------------------|:------------------|:-------------|:----------------------------|
| **Type**           | uint              | uint64       | hex string                  |
| **Description**    | Index type prefix | Block Height | Transaction Type (xxHashed) |
| **Example Value**  | `7`               | `425`        | `45D66Q565F5DEDB[...]`      |

The value stored at the key is the **the compressed list of all events at the given height of a common type**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

## API

The API component provides APIs to access the execution state at different block heights and registers.
See the [API documentation](./api.md) for details on the different APIs that are available.

**API Package documentation**:

* [REST package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/api/rest)
* [GRPC package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/api/grpc)
* [Rosetta package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/api/rosetta)

### Rosetta API

The Rosetta API needs its own documentation because of the amount of components it has that interact with each other.
The main reason for its complexity is that it needs to be able to execute cadence scripts.

#### Contracts

The contracts component keeps track of Flow contracts on the blockchain and provides a method to retrieve the token contract's address, if it exists, from a currency's symbol.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/rosetta/contracts)

#### Scripts

The script package produces Cadence scripts.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/rosetta/scripts)

#### Invoker

This component, given a Cadence script, can execute it at any given height and return the value produced by the script.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/rosetta/invoker)

#### Validator

The Validator components validates whether a given identifier is valid.
It can be used to validate blocks, networks, accounts, transactions and currencies.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/rosetta/validator)

#### Retriever

The retriever uses all the aforementioned components to retrieve account balances, blocks and transactions.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/rosetta/retriever)
