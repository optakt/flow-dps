# Architecture

This document describes the internal components that the Flow Data Provisioning Service is constituted of, as well as the API it exposes.

## Internal Components

### Chain

The chain interface is responsible for reconstructing a view of the sequence of blocks, along with their metadata.
It allows the consumer to step from the root block to the last sealed block, while presenting height, block identifier and state commitment for each step.

TODO: Need a license to be able to link the godoc.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/chain)

#### Filesystem

The Filesystem Chain uses the execution node's on-disk key-value store to reconstruct the block sequence.

#### Network

The Network Chain retrieves data directly from access nodes in order to reconstruct the block sequence.

### Streamer

The streamer interface is responsible for streaming in-order trie updates.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/streamer)

#### Filesystem

The Filesystem Streamer reads trie updates from the LedgerWAL directly.

#### Network

The Network Streamer receives trie updates through its network subscription on the execution node.

### Mapper

The mapper interface is responsible for mapping incoming state trie updates to blocks.
Generally, trie updates come in by chunk, so each block maps from zero to multiple trie updates.
Once a block is mapped to its respective trie updates, the mapper forwards the information to the indexer.

#### Live Mapper

TODO

#### Write-Ahead Log Mapper

TODO

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/mapper)

### Indexer

The indexer interface is responsible for receiving a set of trie updates for each block and creating the necessary main indexes and auxiliary in the on-disk database.
These indexes allow efficient retrieval of the state at arbitrary block heights in the state history.

[Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/indexer)

### Ledger

The ledger interface is responsible for providing clean APIs to access the execution state at different block heights and registers.

**More resources**:

* [API documentation](./api.md)
* [Package documentation](https://pkg.go.dev/github.com/awfm9/flow-dps/ledger)