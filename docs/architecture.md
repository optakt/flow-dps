# Architecture

This document describes the internal components that the Flow Data Provisioning Service is constituted of, as well as the API it exposes.

**Table of Contents**

1. [Chain](#chain)
    1. [Filesystem](#filesystem-chain)
    2. [Network](#network-chain)
2. [Feeder](#feeder)
    1. [Filesystem](#filesystem-feeder)
    2. [Network](#network-feeder)
3. [Mapper](#mapper)
4. [Store](#store)
    1. [Database Schema](#database-schema)
        1. [Block-To-Height Index](#block-to-height-index)
        2. [Commit-To-Height Index](#commit-to-height-index)
        3. [Path Deltas Index](#path-deltas-index)
5. [API](#api)

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

The DPS uses [BadgerDB](https://github.com/dgraph-io/badger) to store datasets of state changes and block information.

#### Block-To-Height Index

In this index, keys map the block ID to the block height, and they are [prefixed with `1`](https://github.com/awfm9/flow-dps/blob/master/model/prefixes.go#L4).

TODO: Replace link to file and line with a link to the godoc that shows those consts.

| **Length** (bytes) | `1`               | `8`        |
|:-------------------|:------------------|:-----------|
| **Type**           | byte              | hex hash   |
| **Description**    | Index type prefix | Block ID   |
| **Example Value**  | `1`               | `1FD5532A` |

The value stored at that key is the **Height** of the referenced block.

##### Commit-To-Height Index

In this index, keys map the commit hash to the block height, and they are [prefixed with `2`](https://github.com/awfm9/flow-dps/blob/master/model/prefixes.go#L5).

| **Length** (bytes) | `1`               | `8`        |
|:-------------------|:------------------|:-----------|
| **Type**           | byte              | hex hash   |
| **Description**    | Index type prefix | Commit     |
| **Example Value**  | `2`               | `3F5D8120` |

The value stored at that key is the **Height** of the referenced commit's block.

##### Path Deltas Index

This last index maps a block ID to all the paths that are changed within its state updates. Its keys are prefixed with [prefixed with `3`](https://github.com/awfm9/flow-dps/blob/master/model/prefixes.go#L6).

| **Length (bytes)** | `1`               | `pathfinder.PathByteSize` | `8`          |
|:-------------------|:------------------|:--------------------------|:-------------|
| **Type**           | uint              |          string           | uint64       |
| **Description**    | Index type prefix |       Register path       | Block Height |
| **Example Value**  | `3`               |      `/0//1//2/uuid`      | `425`        |

The value stored at that key is **the compressed payload of the change at the given path**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

## API

The API component provides APIs to access the execution state at different block heights and registers.
See the [API documentation](./api.md) for details on the different APIs that are available.

**API Package documentation**:

* [REST package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/rest)
* [GRPC package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/grpc)

// TODO: These packages will move to `/api/<name>` once https://github.com/awfm9/flow-dps/pull/23 gets merged.
