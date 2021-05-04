# Flow Data Provisioning Service

The Flow Data Provisioning Service aims at providing a scalable and efficient way to access the history of the Flow
execution state, both for the current live sporks and for past sporks, as well as implementing the Rosetta API
specification, which could serve as the basis for a potential Coinbase integration of Flow.

The index for past sporks is built using the Flow execution node's key-value store (Badger DB) and on-disk write-ahead
log (LedgerWAL).

For the live spork index, the DPS subscribes to an execution node and receives state trie and transaction event updates
over the network.

The Flow DPS maintains multiple specialized indexes for different purposes. One index is used for accessing the entire
execution state at any given height, while another is used to follow the history of a specific Ledger register over time.
Contrary to the execution node's LedgerWAL, the indexes allow random access to the state trie at any block height, which
enables state retrieval at any point in history and far beyond the execution node's pruning limit.

The DPS also supports a set of custom proxy resources for token vaults and staking/delegating resources. This allows the
DPS to track account balances for locked, staked and delegated tokens for accounts which deploy these custom proxy
resources.

## Road Map

| Milestone |                  Description                  | Past Spork State | Past Spork Events | Live Spork State | Live Spork Events | Raw API | Ledger API | Rosetta API | Liquid Balance | Locked Balance | Staked Balance | Delegated Balance | State Verification | State Proofs |
|:---------:|:---------------------------------------------:|:----------------:|-------------------|------------------|-------------------|---------|------------|-------------|----------------|----------------|----------------|-------------------|--------------------|--------------|
|    P.1    |        Past Spork support for registers       |         X        |                   |                  |                   |    X    |      X     |             |        X       |                |                |                   |                    |              |
|    P.2    |         Past Spork support with events        |         X        |         X         |                  |                   |    X    |      X     |             |        X       |                |                |                   |                    |              |
|    R.1    |    Rosetta API support for default balance    |         X        |         X         |                  |                   |    X    |      X     |      X      |        X       |                |                |                   |                    |              |
|    L.1    |        Live Spork support for registers       |         X        |         X         |         X        |                   |    X    |      X     |      X      |        X       |                |                |                   |                    |              |
|    L.2    |         Live Spork support with events        |         X        |         X         |         X        |         X         |    X    |      X     |      X      |        X       |                |                |                   |                    |              |
|    R.2    | Rosetta API support with sub-account balances |         X        |         X         |         X        |         X         |    X    |      X     |      X      |        X       |        X       |        X       |         X         |                    |              |
|    C.1    |   Cryptographic Verification of local state   |         X        |         X         |         X        |         X         |    X    |      X     |      X      |        X       |        X       |        X       |         X         |          X         |              |
|    C.2    |  Cryptographic Proofs for remote state access |         X        |         X         |         X        |         X         |    X    |      X     |      X      |        X       |        X       |        X       |         X         |          X         |       X      |

## Architecture

### Components

socket for live ones.)
The Flow Data Provisioning Service is constituted of three main components.
1. The **Chain** interface is responsible for reconstructing a view of the sequence of blocks, along with their metadata. It allows the consumer to step from the root block to the last sealed block, while presenting height, block identifier and state commitment for each step. The file i/o version does so by using the execution node's on-disk key-value store, while the network version relies on data retrieved from access nodes.
2. The **Streamer** interface is responsible for streaming in-order trie updates from different sources; the file i/o version reads them from the LedgerWAL, while the network version receives trie updates through its network subscription on the execution node.
3. The **Mapper** interface is responsible for mapping incoming state trie updates to blocks. Generally, trie updates come in by chunk, so each block maps from zero to multiple trie updates. Once a block is mapped to its respective trie updates, the mapper forwards the information to the indexer.
4. The **Indexer** interface is responsible for receiving a set of trie updates for each block and creating the necessary main indexes and auxiliary in the on-disk database. These indexes allow efficient retrieval of the state at arbitrary block heights in the state history.
5. The **Ledger** interface is responsible for providing clean APIs to access the execution state at different block heights and registers. Its implementations use the underlying indexes to emulate the same APIs as the Flow execution node Ledger, or the Rosetta Data API.

### Diagram

The following diagram describes a simple overview of an example where the DPS is reading from two sporks. One of them
is a live one, which sends

```text
┌─────────────────┐
│                 │
│   Past Spork    │
│                 │
│                 │
│ ┌─────────────┐ │
│ │             │ │
│ │  Exec Node  │ │
│ │             │ │
│ │ ┌─────────┐ │ │
│ │ │LedgerWAL├─┼─┼───────────────────────────────────┐
│ │ └─────────┘ │ │                                   │
│ │             │ │                                   │
│ │ ┌─────────┐ │ │                                   │
│ │ │Badger DB├─┼─┼─────────────────┐                 │
│ │ └─────────┘ │ │                 │                 │
│ │             │ │                 │                 │
│ └─────────────┘ │                 │                 │
│                 │                 │                 │
└─────────────────┘                 │                 │
                                    ▼                 ▼                  ┌──────────────────┐
                           ┌────────────────┬─────────────────────────┐  │ REST/GRPC Client │
                           │   WALMapper    ◄   Filesystem Chain      │  └────────┬─────────┘
                           ├───────▼▼▼──────┼───────────┬─────────────┤           │
                           │                │           ► Raw API     │◄──────────┘
                           │                │           ├─────────────┤                 ┌─────────────────────────────────┐
                           │    Indexer     ►  Ledger   ► Ledger API  │◄────────────────┤ Get(*ledger.Query) ledger.Value │
                           │                │           ├─────────────┤                 └─────────────────────────────────┘
                           │                │           ► Rosetta API │◄──────────┐
                           ├──────▲▲▲───────┼───────────┴─────────────┤           │
                           │  LiveMapper    ◄     Network Chain       │           │
                           └────────────────┴─────────────────────────┘      ┌────┴─────┐
                                    ▲                 ▲                      │ Rosetta  │
                                    │                 │                      │ Client   │
┌─────────────────┐                 │                 │                      └──────────┘
│                 │                 │                 │
│  Live Network   │                 │                 │
│                 │                 │                 │
│ ┌─────────────┐ │                 │                 │
│ │             │ │ Publish Socket  │                 │
│ │  Exec Node  ├─┼─────────────────┘                 │
│ │             │ │ * trie updates                    │
│ └─────────────┘ │ * transaction events              │
│                 │                                   │
│ ┌─────────────┐ │                                   │
│ │             │ │                                   │
│ │ Access Node │ │                                   │
│ │             │ │                                   │
│ ├─────────────┤ │ Block headers                     │
│ │  GRPC  API  ├─┼───────────────────────────────────┘
│ └─────────────┘ │
│                 │
└─────────────────┘
```
