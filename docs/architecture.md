# Architecture

The following document gives a brief overview of different parts of the Flow DPS architecture.

## Flow DPS Indexer

The Flow DPS Indexer can only index past sporks, contrary to the [Flow DPS Live](#flow-dps-live) binary which indexes live sporks.
In order to build its index, it uses the spork's root checkpoint, and goes through the protocol state database and ledger write-ahead logs of an execution node to index the entire history of the state of the blockchain.
It does not run the DPS API on its own, so the API must be started using the produced index, once the indexing operation is complete.

[![Flow DPS Indexer](./svg/flow_dps_indexer.svg)](./svg/flow_dps_indexer.svg)

### Components

* [Loader](https://pkg.go.dev/github.com/optakt/flow-dps/service/loader) -- Loads root checkpoint and restores the contained execution state trie.
* [Feeder](https://pkg.go.dev/github.com/optakt/flow-dps/service/feeder) -- Loads trie updates from ledger WAL in chronological order.
* [Chain](https://pkg.go.dev/github.com/optakt/flow-dps/service/chain) -- Reads chain data from the protocol state database.
* [Mapper](https://pkg.go.dev/github.com/optakt/flow-dps/service/mapper) -- Orchestrates the aforementioned components to build its index.
* [Indexer](https://pkg.go.dev/github.com/optakt/flow-dps/service/index) -- Exposes read and write access to the index database.

## Flow DPS Live

The DPS Live binary fetches data from the Flow Network in two distinct ways:

* It acts as an [unstaked consensus follower](https://github.com/onflow/full-observer-node-example), which allows it to have access to the protocol state and be notified when a new block is finalized.
* It downloads block execution records from a Google Cloud Storage bucket which is continuously updated by an execution node on the Flow network.

All of this information is then mapped into the DPS index, which is used by the DPS API. The Live binary indexes and serves the API simultaneously.

[![Flow DPS Live](./svg/flow_dps_live.svg)](./svg/flow_dps_live.svg)

### Components

* [GCPStreamer](https://pkg.go.dev/github.com/optakt/flow-dps/service/cloud) -- Downloads block records from a Google Cloud Storage bucket.
* [Consensus Tracker](https://pkg.go.dev/github.com/optakt/flow-dps/service/tracker#Consensus) -- Provides access to the protocol state database of the unstaked consensus follower and to the block execution records of the execution tracker.
* [Execution Tracker](https://pkg.go.dev/github.com/optakt/flow-dps/service/tracker#Execution) -- Reads block execution records from the GCP streamer and provides access to the state trie updates contained therein.
* [Mapper](https://pkg.go.dev/github.com/optakt/flow-dps/service/mapper) -- Uses the aforementioned components to build its index.
* [Indexer](https://pkg.go.dev/github.com/optakt/flow-dps/service/index) -- Exposes a Reader and a Writer which give access to the index database.
* [DPS API](https://pkg.go.dev/github.com/optakt/flow-dps/api/dps) -- Exposes the [DPS API](./dps-api.md), and reads from the DPS index.

## DPS APIs

The DPS API is a GRPC API that allows reading any data that was indexed by the Flow DPS, at any given height.
The DPS API can also serve as the foundation for the [Flow Rosetta API](https://github.com/optakt/flow-dps-rosetta) and the [Flow Access API](https://github.com/optakt/flow-dps-access).

[![DPS APIs](./svg/api.svg)](./svg/api.svg)