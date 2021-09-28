## Index Schema

The DPS uses [BadgerDB](https://github.com/dgraph-io/badger) to store datasets of state changes and block information to build all the indexes required for random protocol and execution state access.

#### First Height

The value under this key keeps track of the first finalized block.

| **Length** (bytes) | `1`               |
|:-------------------|:------------------|
| **Type**           | byte              |
| **Description**    | Index type prefix |
| **Example Value**  | `1`               |

The value stored (only once) is the **height** of the first indexed block.

#### Last Height

The value under this key keeps track of the last finalized block.

| **Length** (bytes) | `1`               |
|:-------------------|:------------------|
| **Type**           | byte              |
| **Description**    | Index type prefix |
| **Example Value**  | `2`               |

The value stored (updated each indexed block) is the **height** of the last indexed block.

#### Header Index

In order to provide an efficient implementation of the Rosetta API, this index maps block heights to block headers.
The header contains the metadata for a block as well as a hash representing the combined payload of the entire block.

| **Length (bytes)** | `1`               | `8`          |
|:-------------------|:------------------|:-------------|
| **Type**           | uint              | uint64       |
| **Description**    | Index type prefix | Block Height |
| **Example Value**  | `3`               | `425`        |

The value stored at that key is the **height** of the referenced state commitment's block.

#### Commit Index

In this index, keys map the block height to the state commitment hash.

| **Length** (bytes) | `1`               | `8`          |
|:-------------------|:------------------|:-------------|
| **Type**           | byte              | uint64       |
| **Description**    | Index type prefix | Block Height |
| **Example Value**  | `4`               | `425`        |

The value stored at that key is the **state commitment** of the referenced block height.

#### Events Index

The events index indexes events grouped by block height and transaction type.
The block height is first in the index so that we can look through all events at a given height regardless of type using a key prefix.

| **Length (bytes)** | `1`               | `8`          | `64`                        |
|:-------------------|:------------------|:-------------|:----------------------------|
| **Type**           | uint              | uint64       | hex string                  |
| **Description**    | Index type prefix | Block Height | Transaction Type (xxHashed) |
| **Example Value**  | `5`               | `425`        | `45D66Q565F5DEDB[...]`      |

The value stored at the key is the **a compressed slice of all events at the given height and given type**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

#### Path Deltas Index

This index maps a block ID to all the paths that are changed within its state updates.

| **Length (bytes)** | `1`               | `pathfinder.PathByteSize` | `8`          |
|:-------------------|:------------------|:--------------------------|:-------------|
| **Type**           | uint              |          string           | uint64       |
| **Description**    | Index type prefix |       Register path       | Block Height |
| **Example Value**  | `6`               |      `/0//1//2/uuid`      | `425`        |

The value stored at that key is **the compressed payload of the payload at the given height and given path**.
It is compressed using [CBOR compression](https://en.wikipedia.org/wiki/CBOR).

#### Block Height Index

In this index, keys map the block IDs to their height.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `7`               | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **block height** of the referenced block ID.

#### Transaction Records

In this record, transactions are mapped by their IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Transaction ID         |
| **Example Value**  | `8`               | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.TransactionBody](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#TransactionBody)** with the referenced ID.

#### Block Transaction Index

In this index, block IDs are mapped to the IDs of the transactions within their block.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `9`               | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded slice of [flow.Identifier](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#Identifier)** for the transactions within the referenced block.

#### Collection Index

In this record, collections are mapped by their IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Collection ID          |
| **Example Value**  | `10`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.LightCollection](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#LightCollection)** with the referenced ID.

#### Collection Guarantee Index

In this record, collections guarantees are mapped by their collection IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Collection ID          |
| **Example Value**  | `10`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.CollectionGuarantee](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#CollectionGuarantee)** with the referenced ID.

#### Block Collection Index

In this index, block IDs are mapped to the IDs of the collections within that block.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Block ID               |
| **Example Value**  | `11`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded slice of [flow.Identifier](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#Identifier)** for the collections within the referenced block.

#### Transaction Result Index

In this index, transaction IDs are mapped to their results.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Transaction ID         |
| **Example Value**  | `12`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.TransactionResult](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#TransactionResult)** for the referenced transaction.

#### Seals Index

In this index, seals are mapped by their IDs.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Seal ID                |
| **Example Value**  | `14`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **CBOR-encoded [flow.Seal](https://pkg.go.dev/github.com/onflow/flow-go/model/flow#Seal)** with the referenced ID.

#### Block Seals Index

In this index, heights are mapped to the IDs of the seals at that height.

| **Length** (bytes) | `1`               | `8`                    |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | uint64                 |
| **Description**    | Index type prefix | Block Height           |
| **Example Value**  | `15`              | `425`                  |

#### Transaction Height Index

In this index, keys map the transaction IDs to their height.

| **Length** (bytes) | `1`               | `64`                   |
|:-------------------|:------------------|:-----------------------|
| **Type**           | byte              | flow.Identifier        |
| **Description**    | Index type prefix | Transaction ID         |
| **Example Value**  | `16`              | `45D66Q565F5DEDB[...]` |

The value stored at that key is the **block height** of the referenced transaction ID.