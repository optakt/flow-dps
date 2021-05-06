# Introduction

This document is aimed at introducing developers to the Flow Data Provisioning Service project.

## Table of Contents

1. [Getting Started](#getting-started)
3. [What is Flow](#flow)
    1. [Glossary](#glossary)
4. [Developer Guide](#developer-guide)
    1. [Installation](#installation)
    2. [Setting up a test environment](#setting-up-a-test-environment)
5. [More resources](#more-resources)

## Getting Started

The Flow Data Provisioning Service (DPS) is a web service that maintains and provides access to the history of the Flow [execution state](#execution-nodes).

The reason for this need is that in the execution state, blocks are pruned after a hundred other blocks have been finalized, which makes it impossible to visualize the history of the chain.
By providing an API which exposes the Flow chain history and implementing the widely used [Rosetta API specification](https://www.rosetta-api.org/), it becomes possible to integrate the Flow blockchain into many 3rd party applications and tools.

When the Flow blockchain is upgraded with a new version that introduces changes in [pathfinding](#merkle-patricia-tries), the old chain gets [_sporked_](#sporks) and a new one gets created based on the state of the old one.
The old version of the chain is maintained as a static chain which will never change, with just one [execution node](#execution-nodes) and one access node to provide read access into the chain's state.

## Flow

Flow is a blockchain that was designed to provide better scalability than other blockchains, by separating the jobs of its nodes into [different roles](#nodes).

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
    - It uses a [BFT](#byzantine-fault) message routing system that guarantees message delivery with a high probability.
- Rewarding and Slashing
    - Flow requires adequate compensation and [slashing](#slashing) mechanics that incentivize nodes to comply with the protocol.
- Honest Stake Fraction
    - Flow requires more than two thirds (2/3) of [stake](#staking) from [Collector](#collector-nodes), [Consensus](#consensus-nodes) and [Verification](#verification-nodes) Nodes to be controlled by honest actors (for each node role separately). A super-majority (two thirds) of honest nodes probabilistically guarantees the safety of the Flow protocol.

### Glossary

#### Nodes

<img alt="node roles" src="https://assets.website-files.com/5f6294c0c7a8cdd643b1c820/5fcff1a16213f9d33a6db5ff_ezgif.com-resize.gif" />

##### Execution Nodes

The Execution Nodes determine the results of transactions when executed in the order determined by the [Consensus Nodes](#consensus-nodes). They are responsible for scaling the computational power of the blockchain.
They produce cryptographic attestations declaring the result of their efforts in the form of **Execution Receipts**.
The receipts can be used to challenge the claims of an Execution Node when they are shown to be incorrect. They are also used to create proofs of the current state of the chain once they are known to be correct.
In order to allow [Verification Nodes](#verification-nodes) to process the execution receipts, computations of a block are split into chunks. Each execution node publishes additional information about each chunk in its execution receipt for the block executed. Each chunk corresponds to a [collection](#collector-nodes).

Errors introduced by the Execution Nodes are always guaranteed to have four critical attributes:

* **Detectable**: A deterministic process has an objectively correct output. Therefore, even a single honest node in the network can detect deterministic faults and prove the error to all other honest nodes by pointing out the part of the process that was executed incorrectly.
* **Attributable**: The output of all deterministic processes in Flow but be signed with the identity of the node that generated those results. As such, any error that has been detected can be clearly attributed to the node(s) that were responsible for that process.
* **Punishable**: All nodes participating in a Flow network, including Execution Nodes, must put up a [stake](#staking) that can be [slashed](#slashing) if they are found to have exhibited [Byzantine behavior](#byzantine-fault). Since all errors in deterministic processes are detectable and attributable, those errors can be reliably punished via [slashing](#slashing).
* **Recoverable**: The system must have a means to undo errors are they are detected. The property services to deter malicious actors from inducing errors that benefit them more than the slashing penalty.

##### Consensus Nodes

The Consensus Nodes work with transaction batches (called [_collections_](#collector-nodes)).
They form blocks from those transaction data digests, are in charge of sealing those blocks, and adjudicate slashing requests from other nodes (for example, claims that an Execution Node has produced incorrect outputs.)
A block contains collection hashes, and a source of randomness which is used to shuffle the transactions before computing them.
Consensus Nodes don't directly compute the transaction order for a block, but they implicitly determine it by specifying all the given inputs to a deterministic algorithm that computes the order.

Since the responsibility to maintain a large state is delegated to specialized nodes, hardware requirements for consensus nodes remain moderate even for high-throughput blockchains.
This design increases decentralization by allowing for higher levels of participation in consensus by individuals with suitable consumer hardware on home internet connections.

When a consensus node receives a guaranteed collection of transactions, it has to run its consensus algorithm to reach an agreement with other nodes over the set of collections to be included in the next block.
A block of the ordered collection that has undergone the complete consensus algorithm is called a _finalized block_.
A block specifies the included transactions as well as the other inputs (randomness seed, etc.) which are required to execute the computation.
It is worth noting that a block in Flow does not include the resulting execution state of the block execution.

In order to Consensus Nodes to seal blocks, they must commit to the execution result of a block after it is executed and verified.

##### Collector Nodes

For the sake of load-balancing, redundancy and [Byzantine resilience](#byzantine-fault), the Collector Nodes are [staked](#staking) equally and randomly positioned into clusters of roughly identical size.
At the beginning of an epoch, each Collection Node is randomly assigned to exactly one cluster.
Each cluster of Collector Nodes acts as a Gateway of Flow with the external world.
This clustering mechanism avoids heterogeneous systems where a Collector Node with better service would be getting all the traffic and end up reducing the decentralization of the whole system as well as starving out other collectors.

External clients submit their transactions to Collector Nodes. Upon receiving well-formed transactions, Collector Nodes introduce them to the rest of their cluster.
The collector nodes of one cluster batch the received transactions into collections. Only a hash reference to a collection is submitted to the consensus nodes for inclusion in a block.

Each cluster of Collector Nodes generates their collections one at a time. Before a new collection is started, the current one has to be closed and sent to the Consensus Nodes for inclusion in a block.
The Collector Nodes' consensus protocol determines when to start/end a collection and which transactions to include in the collection. The result of that consensus is called a _guaranteed collection_.

##### Verification Nodes

The Verification Nodes are in charge of collectively verifying the correctness of the Execution Nodes' published results.
With the chunking approach of Flow, each node only checks a small fraction of chunks.
A verification Node requests the information it needs for re-computing the chunks it is checking from the Execution Nodes.
It approves the result of a chunk by publishing a _result approval_ for that chunk.

##### Access Nodes

Access Nodes are part of the network **around** the core network, which provides services to scale more easily.
They have a copy of the entire state of the Execution Nodes, and provide a service to see changes in the Execution State.

The way they are able to read from the state of the Execution Nodes is by sending them requests to execute [Cadence](#cadence) Scripts which read from the state and sends back the results, which is then forwarded to the SDK client by Access Nodes.
This is highly inefficient because Access Nodes have to proxy all the requests, and also do so by remotely executing Cadence Scripts, which brings a lot of overhead.

#### Proof of Stake

Proof of Stake (PoS) protocols are a class of consensus mechanisms for blockchains that work by selecting validators in proportion to their [](#staking) in the associated cryptocurrency.

[More information](https://en.wikipedia.org/wiki/Proof_of_stake)

#### Staking

A node in Flow is required to deposit some stake in order to run a role. This requires the node to submit a staking transaction.
The staking transactions for the next epoch takes place before a specific deadline in the current epoch.
Once the staking transaction is processed by the [Execution Nodes](#execution-nodes), the stake is withdrawn from the node's account balance and is explicitly recorded in the _Execution Receipt_.
Upon [Consensus Nodes](#consensus-nodes) sealing the block that contains this staking transaction, they update the protocol state affected by the transaction, and publish the corresponding staking update in the block that holds the seal.
Staked nodes are compensated through both block rewards and transaction fees and all roles require a minimum stake to formally participate in that role.

To stake, an actor submits a staking transaction which includes its public staking key.
Once the staking transactions are included in a block and executed by the Execution Nodes, a notification is embedded into the corresponding Execution Receipt.
When sealing the execution result, the Consensus Nodes will update the protocol state of the staking nodes accordingly.

For unstaking, a node submits a transaction signed by its staking key. Once an unstaking transaction is included in a block during an epoch, it discharges the associated node’s protocol state as of the following epoch.
The discharged stake of an unstaked node is effectively maintained on hold, i.e., it can be slashed, but it is not returned to the unstaked node’s account.
The stake is returned to the unstaked node after a waiting period of at least one epoch. The reason for doing so is two-fold.
First, detecting and adjudicating protocol violations might require some time. Hence, some delay is required to ensure that there is enough time to slash a misbehaving node before its stake is refunded.
Second, to prevent a long-range attack wherein a node unstakes, and then retroactively misbehaves, e.g., a Consensus Node signing an alternative blockchain to fork the protocol.

#### Slashing

Any [staked](#staking) node of Flow can detect and attribute misbehavior to another staked node that committed it. Upon detecting and attributing misbehavior, the node issues a slashing challenge against the faulty node.
Slashing challenges are submitted to the [Consensus Nodes](#consensus-nodes). The slashing challenge is a request for slashing a staked node du to misbehavior and derivation from the protocol.
As the sole entity of the system responsible for updating the protocol state, Consensus Nodes adjudicate slashing challenges and adjust the protocol state (staking balances) of the faulty nodes accordingly.
Based of the result of adjudication, the protocol state (i.e, the stake) of a node may be slashed within an epoch.
A block's protocol state can be altered after it has been approved, in which case changes in the protocol state of a block propagate to the children of this block.

#### Sporks

Currently, every couple of weeks, the network is turned off, updated and turned on again. This process is called a Spork.

More information](https://docs.onflow.org/node-operation/spork)

#### Cadence

Cadence is a resource-oriented programming language that introduces new features to smart contract programming that help developers ensure that their code is safe, secure, clear and approachable. Some of these features are:

* Type safety and a strong static type system
* Resource-oriented programming, a new paradigm that pairs linear types with object capabilities to create a secure and declarative model for digital ownership by ensuring that resources (and their associated assets) can only exist in one location at a time, cannot be copied, and cannot be accidentally lost or deleted
* Built-in pre-conditions and post-conditions for functions and transactions
* The utilization of capability-based security, which enforces access control by requiring that access to objects is restricted to only the owner and those who have a valid reference to the object

[More information](https://docs.onflow.org/cadence)

#### Byzantine Fault

A Byzantine fault is a condition of a computer system, particularly distributed computing systems, where components may fail and there is imperfect information on whether a component has failed.
In a Byzantine fault, a component such as a server can inconsistently appear both failed and functioning to failure-detection systems, presenting different symptoms to different observers.
It is difficult for the other components to declare it failed and shut it out of the network, because they need to first reach a consensus regarding which component has failed in the first place.

[More information](https://en.wikipedia.org/wiki/Byzantine_fault)

#### Merkle Patricia Tries

A Merkle Patricia Trie is a [radix tree](https://en.wikipedia.org/wiki/Radix_tree) with a few modifications.
In a normal radix tree, a key is the actual path taken through the tree to get to the corresponding value.
The Flow implementation of radix trees introduces a number of improvements:

* To make the tree cryptographically secure, each node is referenced by its hash. With this scheme, the root node becomes a cryptographic fingerprint of the entire data structure (this is the _Merkle_ part)
* Multiple node types are introduced to improve efficiency. There are blank nodes, leaf nodes (which are a list of keys and values), but also extension nodes which have key/value pairs which point to other nodes.
* There are also branch nodes, which are arrays of 17 elements of which each is a one of the hexadecimal characters and points to other nodes (plus one k/v pair at the end in case the path has been fully traversed).

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

If you want to generate smaller checkpoints and generate them quicker, you can edit the generated `docker-compose.nodes.yml` and add the following argument to the [Execution Nodes](#execution-nodes): `--checkpoint-distance=1`.
Another recommended tweak is to edit the `SegmentSize` constant from `32 * 1024 * 1024` to simply `32 * 1024`. You can find this constant variable in `ledger/complete/wal/wal.go`.

Once you are happy with your configuration, you can run the local network by running `make start`.

Now, the local network is running, but nothing is happening since there are no transactions and accounts being registered on it.
You can then use [`flow-sim`](https://github.com/awfm9/flow-sim) to create fake activity on your test network.
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