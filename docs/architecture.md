# Architecture

This document describes the internal components that the Flow Data Provisioning Service is constituted of, as well as
the API it exposes.

**Table of Contents**

1. [Chain](#chain)
    1. [Disk Chain](#disk-chain)
2. [Feeder](#feeder)
    1. [Disk Feeder](#disk-feeder)
3. [Mapper](#mapper)
4. [Index](#index)
    1. [Database Schema](#database-schema)
        1. [First Height](#first-height)
        2. [Last Height](#last-height)
        3. [Header Index](#header-index)
        4. [Commit Index](#commit-index)
        5. [Events Index](#events-index)
        6. [Path Deltas Index](#path-deltas-index)
        7. [Block Height Index](#block-height-index)
        8. [Transaction Records](#transaction-records)
        9. [Block Transaction Index](#block-transaction-index)
        10. [Collection Transaction Index](#collection-transaction-index)
        11. [Block Collection Index](#block-collection-index)
        12. [Transaction Result Index](#transaction-result-index)
        13. [Seals Index](#seals-index)
        14. [Block Seals Index](#block-seals-index)
        15. [Sealed Height](#sealed-height)
        16. [Transaction Height Index](#transaction-height-index)

## Chain

The Chain component is responsible for reconstructing a view of the sequence of blocks, along with their metadata.
It allows the consumer to step from the root block to the last sealed block, while providing data related to each height along the sequence of blocks, such as block identifier, state commitment and events.
It is used by the [Mapper](#mapper) to map a set of deltas from the [Feeder](#feeder) to each block height.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/service/chain)

### Disk Chain

The [Disk Chain](https://pkg.go.dev/github.com/optakt/flow-dps/service/chain#Disk) uses the execution node's on-disk key-value store for the Flow protocol state to reconstruct the block sequence.

## Feeder

The Feeder component is responsible for streaming trie updates to the [Mapper](#mapper).
It outputs a state delta for each requested state commitment, so that the [Mapper](#mapper) can follow the sequence of changes to the state trie and attribute each change to a block height.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/service/feeder)

### Disk Feeder

The [Disk Feeder](https://pkg.go.dev/github.com/optakt/flow-dps/service/feeder#Disk) reads trie updates directly from an on-disk write-ahead log of the execution node.

## Mapper

The Mapper component is at the core of the DPS. It is responsible for mapping incoming state trie updates to blocks.
In order to do that, it depends on the [Feeder](#feeder) and [Chain](#chain) components to get state trie updates and block information, as well as on the [Index](#index) component for indexing.
Generally, trie updates come in by chunk, so each block height corresponds to an arbitrary number of trie updates, from zero to many.
Once a block height is mapped to its respective trie updates, the mapper uses the indexer to persist the information.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/service/mapper)

## Index

The Index component has a [Index Writer](https://pkg.go.dev/github.com/optakt/flow-dps/index#Writer), responsible for indexing the data at each block height.
The writer creates a number of auxiliary indexes that allow us to access the state of each register at any block height.
This index is then accessed by the [Index Reader](https://pkg.go.dev/github.com/optakt/flow-dps/index#Reader) to retrieve block data.
The reader serves as an intermediary to the Flow Virtual Machine, allowing execution of Cadence scripts on top of its data.
Additionally, it provides access to the DPS API through the GRPC server, which in turn allows remote clients to execute scripts as well.

[Package documentation](https://pkg.go.dev/github.com/optakt/flow-dps/service/index)

### Database Schema

The DPS uses [BadgerDB](https://github.com/dgraph-io/badger) to store datasets of state changes and block information to build all the indexes required for random protocol and execution state access.
It does not re-use any of the protocol state database, but instead re-indexes everything, so that all databases used to bootstrap the index can be discarded subsequently.

##### First Height

The value under this key keeps track of the first finalized block.

| **Length** (bytes) | `1`               |
|:-------------------|:------------------|
| **Type**           | byte              |
| **Description**    | Index type prefix |
| **Example Value**  | `1`               |

The value stored (only once) is the **height** of the first indexed block.

##### Last Height

The value under this key keeps track of the last finalized block.

| **Length** (bytes) | `1`               |
|:-------------------|:------------------|
| **Type**           | byte              |
| **Description**    | Index type prefix |
| **Example Value**  | `2`               |

The value stored (updated each indexed block) is the **height** of the last indexed block.

##### Header Index

In order to provide an efficient implementation of the Rosetta API, this index maps block heights to block headers.
The header contains the metadata for a block as well as a hash representing the combined payload of the entire block.

| **Length (bytes)** | `1`               | `8`          |
|:-------------------|:------------------|:-------------|
| **Type**           | uint              | uint64       |
| **Description**    | Index type prefix | Block Height |
| **Example Value**  | `3`               | `425`        |

The value stored at that key is the **height** of the referenced state commitment's block.

##### Commit Index

In this index, keys map the block height to the state commitment hash.

| **Length** (bytes) | `1`               | `8`          |
|:-------------------|:------------------|:-------------|
| **Type**           | byte              | uint64       |
| **Description**    | Index type prefix | Block Height |
| **Example Value**  | `4`               | `425`        |

The value stored at that key is the **state commitment** of the referenced block height.

##### Events Index

The events index indexes events grouped by block height and transaction type.
The block height is first in the index so that we can look through all events at a given height regardless of type using a key prefix.

| **Length (bytes)** | `1`               | `8`          | `64`                        |
|:-------------------|:------------------|:-------------|:----------------------------|
| **Type**           | uint              | uint64       | hex string                  |
| **Description**    | Index type prefix | Block Height | Transaction Type (xxHashed) |
| **Example Value**  | `5`               | `425`        | `45D66Q565F5DEDB[...]`      |

The value stored at the key is the **a compressed slice of all events at the given height and given type**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

##### Path Deltas Index

This index maps a block ID to all the paths that are changed within its state updates.

| **Length (bytes)** | `1`               | `pathfinder.PathByteSize` | `8`          |
|:-------------------|:------------------|:--------------------------|:-------------|
| **Type**           | uint              |          string           | uint64       |
| **Description**    | Index type prefix |       Register path       | Block Height |
| **Example Value**  | `6`               |      `/0//1//2/uuid`      | `425`        |

The value stored at that key is **the compressed payload of the payload at the given height and given path**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

##### Block Height Index

In this index, keys map the block IDs to their height.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `7`               | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **block height** of the referenced block ID.

##### Transaction Records

In this record, transactions are mapped by their IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Transaction ID         |
| **Example Value**  | `8`               | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.Transaction](https://pkg.go.dev/github.com/onflow/model/flow#Transaction)** with the referenced ID.

##### Block Transaction Index

In this index, block IDs are mapped to the IDs of the transactions within their block.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `9`               | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded slice of [flow.Identifier](https://pkg.go.dev/github.com/onflow/model/flow#Identifier)** for the transactions within the referenced block.

##### Collection Transaction Index

In this record, collections are mapped by their IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Collection ID         |
| **Example Value**  | `10`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.LightCollection](https://pkg.go.dev/github.com/onflow/model/flow#LightCollection)** with the referenced ID.

##### Block Collection Index

In this index, block IDs are mapped to the IDs of the collections within their block.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `11`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded slice of [flow.Identifier](https://pkg.go.dev/github.com/onflow/model/flow#Identifier)** for the collections within the referenced block.

##### Transaction Result Index

In this index, transaction IDs are mapped to their results.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Transaction ID         |
| **Example Value**  | `12`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.TransactionResult](https://pkg.go.dev/github.com/onflow/model/flow#TransactionResult)** for the referenced transaction.

##### Seals Index

In this index, seals are mapped by their IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Seal ID                |
| **Example Value**  | `14`              | `45D66Q565F5DEDB[...]` |

##### Block Seals Index

In this index, block IDs are mapped to the IDs of the seals within that block.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `15`              | `45D66Q565F5DEDB[...]` |

##### Transaction Height Index

In this index, keys map the transaction IDs to their height.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Transaction ID         |
| **Example Value**  | `16`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **block height** of the referenced transaction ID.

##### Sealed Height

The value under this key keeps track of the last sealed block.

| **Length** (bytes) | `1`               |
|:-------------------|:------------------|
| **Type**           | byte              |
| **Description**    | Index type prefix |
| **Example Value**  | `17`              |

The value stored (updated each indexed block) is the **height** of the last sealed block.