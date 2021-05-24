# Introduction

This document is aimed at introducing developers to the Flow Data Provisioning Service project.

**Table of Contents**

1. [Getting Started](#getting-started)
2. [Flow](#flow)
   1. [Glossary](#glossary)
      1. [Nodes](#nodes)
         1. [Execution Nodes](#execution-nodes)
         2. [Consensus Nodes](#consensus-nodes)
         3. [Collection Nodes](#collection-nodes)
         4. [Verification Nodes](#verification-nodes)
         5. [Access Nodes](#access-nodes)
      2. [Proof of Stake](#proof-of-stake)
      3. [Staking](#staking)
      4. [Slashing](#slashing)
      5. [Sporks](#sporks)
      6. [Cadence](#cadence)
      7. [Transactions](#transactions)
      8. [Byzantine Fault](#byzantine-fault)
      9. [Merkle Patricia Tries](#merkle-patricia-tries)
   2. [Developer Guide](#developer-guide)
      1. [Installation](#installation)
         1. [Dependencies](#dependencies)
         2. [Manual Build](#manual-build)
      2. [Setting up a test environment](#setting-up-a-test-environment)
   3. [More Resources](#more-resources)

## Getting Started

The Flow Data Provisioning Service (DPS) is a web service that maintains and provides access to the history of the Flow [execution state](#execution-nodes).

The reason for this need is that the in-memory execution state is pruned after 300 chunks, which makes it impossible to access the state history.
Also, script execution is currently proxied from the [access nodes](#access-nodes) to [execution nodes](#execution-nodes), which is not scalable.
The DPS makes access to the execution state _available_ (at any block height) and _scalable_ (so it does not infer load on the network nodes).
Doing so also makes it possible to provide an API which exposes the Flow chain history and implements the widely used [Rosetta API specification](https://www.rosetta-api.org/), which allows many 3rd party developers to integrate the Flow blockchain into their applications and tools.

Flow is often [upgraded with breaking changes](#sporks) that require a network restart. The new network with the updated version is started from a snapshot of the previous execution state.
The final version of the previous execution state remains available through a legacy access node that connects to a legacy [execution node](#execution-nodes), but once again this is limited to the last 300 chunks.

## Flow

Flow was designed to provide a blockchain that can scale while preserving composability in a single blockchain state. This is achieved by introducing several [node roles](#nodes), each with their own trade-off for the [scalability trilemma](https://aakash-111.medium.com/the-scalability-trilemma-in-blockchain-75fb57f646df).

Flow is based upon the following set of rules:

- Consensus and Authentication
    - All nodes participating in the system are known to each other.
    - Each node is authenticated through its unforgeable digital signature.
    - Consensus is based on [Proof of Stake](#proof-of-stake)
- Participation in Network
    - The evolution of the chain consists of fixed intervals called epochs
    - To participate in the network, a node must put up the minimum required [stake](#staking) for that role in a specific epoch.
    - A node may participate over multiple epochs.
- Source of randomness
    - Flow requires a reliable source of randomness for seeding its pseudo-RNG
    - The source of randomness enables each seed to be unpredictable by any individual node until the seed itself is generated and published in a decentralized manner
    - This is done using the Distributed Random Beacon (DRB) protocol to generate a fully decentralized, reliable source of randomness.
- Cryptography Primitives
    - Flow requires an aggregatable and non-interactive signature scheme, such as [BLS](https://en.wikipedia.org/wiki/BLS_digital_signature)
- Network Model
    - Flow operates on a partially synchronous network.
- Rewarding and Slashing
    - Flow requires adequate compensation and [slashing](#slashing) mechanics that incentivize nodes to comply with the protocol.
- Honest Stake Fraction
    - Flow requires more than two thirds (2/3) of [stake](#staking) from [collection](#collection-nodes), [consensus](#consensus-nodes) and [verification](#verification-nodes) nodes to be controlled by honest actors (for each node role separately). A super-majority (two thirds) of honest nodes probabilistically guarantees the safety of the Flow protocol.

### Glossary

#### Nodes

<img alt="node roles" src="https://assets.website-files.com/5f6294c0c7a8cdd643b1c820/5fcff1a16213f9d33a6db5ff_ezgif.com-resize.gif" />

##### Execution Nodes

The execution nodes determine the results of [transactions](#transactions) when executed in the order determined by the [consensus nodes](#consensus-nodes).
They are responsible for scaling the computational power of the blockchain, and are the only node role to have access to the **execution state**.
They produce cryptographic attestations declaring the result of their efforts in the form of **execution receipts**.
The receipts can be used to challenge the claims of an execution node when they are shown to be incorrect.
They are also used to create proofs of the current state of the chain once they are known to be correct.
In order to allow [verification nodes](#verification-nodes) to process the execution receipts, computations of a block are split into chunks.
Each execution node publishes additional information about each chunk in its execution receipt for the block executed.
Each chunk corresponds to a [collection](#collection-nodes).

Errors introduced by the execution nodes are always guaranteed to have four critical attributes:

* **Detectable**: A deterministic process has an objectively correct output. Therefore, even a single honest node in the network can detect deterministic faults and prove the error to all other honest nodes by pointing out the part of the process that was executed incorrectly.
* **Attributable**: The output of all deterministic processes in Flow but be signed with the identity of the node that generated those results. As such, any error that has been detected can be clearly attributed to the node(s) that were responsible for that process.
* **Punishable**: All nodes participating in a Flow network, including execution nodes, must put up a [stake](#staking) that can be [slashed](#slashing) if they are found to have exhibited [Byzantine behavior](#byzantine-fault). Since all errors in deterministic processes are detectable and attributable, those errors can be reliably punished via [slashing](#slashing).
* **Recoverable**: The system must have a means to undo errors are they are detected. The property services to deter malicious actors from inducing errors that benefit them more than the slashing penalty.

##### Consensus Nodes

The consensus nodes work with transaction batches (called [_collections_](#collection-nodes)).
They form blocks from collections, are in charge of sealing those blocks, and adjudicate slashing requests from other nodes (for example, claims that an execution node has produced incorrect outputs.)
A block contains collection hashes, and a source of randomness which is used to shuffle the transactions before computing them.
consensus nodes don't directly compute the transaction order for a block, but they implicitly determine it by specifying all the given inputs to a deterministic algorithm that computes the order.

Since the responsibility to maintain a large state is delegated to specialized nodes, hardware requirements for consensus nodes remain moderate even for high-throughput blockchains.
As opposed to other node roles, consensus nodes deal with subjective problems, where there is no single correct answer.
Instead, one answer must be selected through mutual agreement, which is why it is critical for consensus nodes to be numerous and decentralized, and less so for other nodes.
This design increases decentralization by allowing for higher levels of participation in consensus by individuals with suitable consumer hardware on home internet connections.

When a consensus node receives a guaranteed collection of transactions, it has to run its consensus algorithm to reach an agreement with other nodes over the set of collections to be included in the next block.
A block of the ordered collection that has undergone the complete consensus algorithm is called a _finalized block_.
A block specifies the included collections as well as the other inputs (randomness seed, etc.) which are required to execute the computation.
It is worth noting that a block in Flow does not include the resulting execution state of the block execution.

In order for consensus nodes to seal blocks, they must commit to the execution result of a block after it is executed and verified.

##### Collection Nodes

For the sake of load-balancing, redundancy and [Byzantine resilience](#byzantine-fault), the collection nodes are [staked](#staking) equally and randomly positioned into clusters of roughly identical size.
At the beginning of an epoch, each collection node is randomly assigned to exactly one cluster.
Each cluster of collection nodes acts as a gateway to Flow, from the external world.
This clustering mechanism avoids heterogeneous systems where a collection node with better service would be getting all the traffic and end up reducing the decentralization of the whole system as well as starving out other collections.

External clients submit their transactions to collection nodes. Upon receiving well-formed transactions, collection nodes introduce them to the rest of their cluster.
The collection nodes of one cluster batch the received transactions into collections. Only a hash reference to a collection is submitted to the consensus nodes for inclusion in a block.

Each cluster of collection nodes generates their collections one at a time. Before a new collection is started, the current one has to be closed and sent to the consensus nodes for inclusion in a block.
The collection nodes' consensus protocol determines when to start/end a collection and which transactions to include in the collection. The result of that consensus is called a _guaranteed collection_.

##### Verification Nodes

The verification nodes are in charge of collectively verifying the correctness of the execution nodes' published results.
With the chunking approach of Flow, each node only checks a small fraction of chunks.
A verification node requests the information it needs for re-computing the chunks it is checking from the execution nodes.
It approves the result of a chunk by publishing a _result approval_ for that chunk.

##### Access Nodes

Access nodes are part of the network **around** the core network, which provides services to scale more easily, but they are also part of the core network in the sense that they are part of the node identity table and can connect to other nodes of the network directly.

The way they are able to read from the state of the execution nodes is by sending them requests to execute [Cadence](#cadence) scripts which read from the state and sends back the results, which is then forwarded to the SDK client by access nodes.
This is highly inefficient because access nodes have to proxy all the requests, and also do so by sending requests to execution nodes to execute cadence scripts, which adds extra load on the execution nodes.

#### Proof of Stake

Proof of Stake (PoS) protocols are a class of consensus mechanisms for blockchains that work by selecting validators in proportion to their [](#staking) in the associated cryptocurrency.

[More information](https://en.wikipedia.org/wiki/Proof_of_stake)

#### Staking

A node in Flow is required to deposit some stake in order to run a role. This requires the node to submit a staking transaction.
The staking transactions for the next epoch takes place before a specific deadline in the current epoch.
Once the staking transaction is processed by the [execution nodes](#execution-nodes), the stake is withdrawn from the node's account balance and is explicitly recorded in the _execution Receipt_.
Upon [consensus nodes](#consensus-nodes) sealing the block that contains this staking transaction, they update the protocol state affected by the transaction, and publish the corresponding staking update in the block that holds the seal.
Staked nodes are compensated through both block rewards and transaction fees and all roles require a minimum stake to formally participate in that role.

To stake, an actor submits a staking transaction which includes its public staking key.
Once the staking transactions are included in a block and executed by the execution nodes, a notification is embedded into the corresponding execution Receipt.
When sealing the execution result, the consensus nodes will update the protocol state of the staking nodes accordingly.

For unstaking, a node submits a transaction signed by its staking key. Once an unstaking transaction is included in a block during an epoch, it discharges the associated node’s protocol state as of the following epoch.
The discharged stake of an unstaked node is effectively maintained on hold, i.e., it can be slashed, but it is not returned to the unstaked node’s account.
The stake is returned to the unstaked node after a waiting period of at least one epoch. The reason for doing so is two-fold.
First, detecting and adjudicating protocol violations might require some time. Hence, some delay is required to ensure that there is enough time to slash a misbehaving node before its stake is refunded.
Second, to prevent a long-range attack wherein a node unstakes, and then retroactively misbehaves, e.g., a consensus node signing an alternative blockchain to fork the protocol.

#### Slashing

Any [staked](#staking) node of Flow can detect and attribute misbehavior to another staked node that committed it. Upon detecting and attributing misbehavior, the node issues a slashing challenge against the faulty node.
Slashing challenges are submitted to the [consensus nodes](#consensus-nodes). The slashing challenge is a request for slashing a staked node du to misbehavior and derivation from the protocol.
As the sole entity of the system responsible for updating the protocol state, consensus nodes adjudicate slashing challenges and adjust the protocol state (staking balances) of the faulty nodes accordingly.
Based of the result of adjudication, the protocol state (i.e, the stake) of a node may be slashed within an epoch.
A block's protocol state can be altered after it has been approved, in which case changes in the protocol state of a block propagate to the children of this block.

#### Sporks

Currently, every couple of weeks, the network is turned off, updated and turned on again. This process is called a Spork.

[More information](https://docs.onflow.org/node-operation/spork)

#### Cadence

Cadence is a resource-oriented programming language that introduces new features to smart contract programming that help developers ensure that their code is safe, secure, clear and approachable. Some of these features are:

* Type safety and a strong static type system
* Resource-oriented programming, a new paradigm that pairs linear types with object capabilities to create a secure and declarative model for digital ownership by ensuring that resources (and their associated assets) can only exist in one location at a time, cannot be copied, and cannot be accidentally lost or deleted
* Built-in pre-conditions and post-conditions for functions and transactions
* The utilization of capability-based security, which enforces access control by requiring that access to objects is restricted to only the owner, and those who have a valid reference to the object

[More information](https://docs.onflow.org/cadence)

#### Transactions

Transactions are function calls and small programs executed on top of the execution state.

#### Byzantine Fault

A Byzantine fault is a condition of a computer system, particularly distributed computing systems, where components may fail and there is imperfect information on whether a component has failed.
In a Byzantine fault, a component such as a server can inconsistently appear both failed and functioning to failure-detection systems, presenting different symptoms to different observers.
It is difficult for the other components to declare it failed and shut it out of the network, because they need to first reach a consensus regarding which component has failed in the first place.

[More information](https://en.wikipedia.org/wiki/Byzantine_fault)

#### Merkle Patricia Tries

A Merkle Patricia Trie is a [radix tree](https://en.wikipedia.org/wiki/Radix_tree) with a few modifications.
In a normal radix tree, a key is the actual path taken through the tree to get to the corresponding value.
The Flow implementation of radix trees introduces a number of improvements:

* Every leaf node is represented by the hash of the data contained within it. For every other node, we can obtain its hash by hashing its children hashes together, all the way up through the tree until we reach the hash of the root node. This allows us to uniquely represent the entire state with a single state commitment, the root hash. Additionally, we can construct so-called merkle proofs that allow us to cryptographically prove the inclusion of an arbitrary part of the trie in a state represented by its root hash.
* Multiple node types are introduced to improve efficiency. There are blank nodes, leaf nodes (which are a list of keys and values), but also extension nodes which have key/value pairs which point to other nodes.

[More information](https://eth.wiki/fundamentals/patricia-tree)

### Developer Guide

This guide lists the necessary step to get started with installing and testing the Flow DPS.

#### Installation

##### Dependencies

* `go1.16`

##### Manual Build

* `go build main.go`

#### Setting up a test environment

In order to set up a test environment, it is recommended to use [Flow's integration tests](https://github.com/onflow/flow-go/tree/master/integration/localnet).

The first step is to install `flow-go` by following [this documentation](https://github.com/onflow/flow-go#installation) up until running `make install-tools`.

Then, you can head into the `integration/localnet` directory, and run `make init`. This will generate the necessary files to build and run nodes into a local Flow network.

If you want to generate smaller checkpoints and generate them quicker, you can edit the generated `docker-compose.nodes.yml` and add the following argument to the [execution nodes](#execution-nodes): `--checkpoint-distance=1`.
Another recommended tweak is to edit the `SegmentSize` constant from `32 * 1024 * 1024` to simply `32 * 1024`. You can find this constant variable in `ledger/complete/wal/wal.go`.

Once you are happy with your configuration, you can run the local network by running `make start`.

Now, the local network is running, but nothing is happening since there are no transactions and accounts being registered on it.
You can then use [`flow-sim`](https://github.com/optakt/flow-sim) to create fake activity on your test network.
Simply clone the repository and run `go run main.go` and it should automatically start making transaction requests to your local network.

If you just need a valid checkpoint, you can monitor the state that your test network generates by running `watch ls data/consensus/<NodeID>` and waiting until you can see a file named `checkpoint.00000001` appear.

You can then copy part of this data folder to be used in DPS:

* `data/consensus/NodeID` can be given to the DPS as `data`
* `trie/execution/NodeID` can be given as `trie`
* `data/consensus/NodeID/checkpoint.00000001` can be given as `root.checkpoint`

You can then run the DPS, and it should properly build its index based on the given information.

### More Resources

* [Flow Technical Papers](https://www.onflow.org/technical-paper)
* [Flow Developer Documentation](https://docs.onflow.org/)
* [Flow Developer Discord Server](https://onflow.org/discord)
