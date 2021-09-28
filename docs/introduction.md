# Introduction

This document is aimed at introducing developers to the Flow Data Provisioning Service project.

**Table of Contents**

1. [Getting Started](#getting-started)
   1. [Indexing Past Sporks](#indexing-past-sporks)
   2. [Indexing Live Sporks](#indexing-live-sporks)
   3. [Serving Other APIs](#serving-other-apis)
2. [Developer Guide](#developer-guide)
   1. [Installation](#installation)
      1. [Dependencies](#dependencies)
      2. [Build](#build)
   2. [Setting up a test environment](#setting-up-a-test-environment)
3. [More Resources](#more-resources)

## Getting Started

The Flow Data Provisioning Service (DPS) is a service that maintains and provides access to the history of the Flow execution state.

The reason for this need is that the in-memory execution state is pruned after 300 chunks, which makes it impossible to access the state history.
Also, script execution is currently proxied from the access nodes to execution nodes, which is not scalable.
The DPS makes access to the execution state _available_ (at any block height) and _scalable_ (so it does not increase load on the core network).

Flow is often upgraded with breaking changes that require a network restart. The new network with the updated version is started from a snapshot of the previous execution state.
The final version of the previous execution state remains available through a legacy access node that connects to a legacy execution node, but once again this is limited to the last 300 chunks.

### Indexing Past Sporks

In order to index past sporks using Flow DPS, three elements are needed from the spork to be indexed:

* Its root checkpoint;
* The protocol state database from one of its execution nodes; and
* The write-ahead log from one of its execution nodes.

You can then run the `flow-dps-indexer` binary, giving it access to those elements like such:

```console
$ ./flow-dps-indexer -a -l debug -d /var/flow/data/protocol -t /var/flow/data/execution -c /var/flow/bootstrap/root.checkpoint -i /var/flow/data/index
```

Creating the index database can take a very long time and requires large amounts of available storage space and memory.
Once the indexing process is over, you can use the created DPS index to serve the DPS API, by running `flow-dps-server` and giving it access to the index.

```console
$ ./flow-dps-server -i /var/flow/data/index -a localhost:5005
```

You should now have the DPS API available at `localhost:5005`.
It can be used in conjunction with the [Flow Rosetta API](https://github.com/optakt/flow-dps-rosetta) and the [Flow Access API](https://github.com/optakt/flow-dps-access), which both need the address to your DPS API in order to function.

### Indexing Live Sporks

The DPS Live binary handles both indexing and serving the DPS API for its index.
This is because it is impossible for one process to write to the index while another reads from it without causing concurrency issues.

It needs to connect to the Flow network by acting as an unstaked consensus follower, and needs the following information:

* The spork's root checkpoint
* The spork's public bootstrap information file
* The name of the GCS bucket in which block records are available
* The address of the seed node to follow unstaked consensus
* The hex-encoded public network key of the seed node to follow unstaked consensus

The Live Indexer configures the unstaked consensus follower to create its protocol state database at the given location (specified using the `-d` option), and also reads from it to retrieve protocol state data.

```console
$ ./flow-dps-live -u flow-block-data -i /var/flow/index -d /var/flow/data -c /var/flow/bootstrap/root.checkpoint -b /var/flow/bootstrap/public --seed-address access.canary.nodes.onflow.org:9000 --seed-key cfce845fa9b0fb38402640f997233546b10fec3f910bf866c43a0db58ab6a1e4
```

### Serving Other APIs

When using the Live Indexer, the DPS API is already exposed parallel to the indexing. When using the `flow-dps-indexer` however, the indexing needs to be completed before the index can be used to start the DPS API using the `flow-dps-server` binary:

```console
$ ./flow-dps-server -i /var/flow/data/index -a 172.17.0.1:5005
```

Once the API is running, it can be used to serve other APIs as well.

See the documentation of the [Flow Rosetta API](https://github.com/optakt/flow-dps-rosetta) in order to build its binary, and then run the following command:

```console
$ ./flow-rosetta-server -a "172.17.0.1:5005" -p 8080
```

The `-a` argument is used to specify the address of the DPS API, and the `-p` sets the port on which the Rosetta API listens.

Similarly, the [Flow Access API](https://github.com/optakt/flow-dps-access) can be run like such:

```console
$ ./dps-access-api -a "172.17.0.1:5005" -p 5006
```

Just like with the Rosetta API, the `-a` argument is used to specify the address of the DPS API, and the `-p` sets the port on which the Access API listens.

## Developer Guide

This guide lists the necessary step to get started with installing and testing the Flow DPS.

### Installation

#### Dependencies

Go `v1.16` or higher is required to compile `flow-dps`.
Only Linux amd64 builds are supported, because of the dependency to the [`flow-go/crypto`](https://github.com/onflow/flow-go/tree/master/crypto) package.
Please note that it is also required to make sure that your `GOPATH` is exported in your environment in order to generate the DPS API.

If you want to make changes to the GRPC API, the following dependencies are required as well.

* [`protoc`](https://grpc.io/docs/protoc-installation/) version `3.17.3`
* `go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26`
* `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1`
* `go install github.com/srikrsna/protoc-gen-gotag@v0.6.1`

Once they are installed, you can run `go generate ./...` from the root of this repository to update the generated protobuf files.

In order to build the live binary, the following extra steps and dependencies are required:

* [`CMake`](https://cmake.org/install/)

Please note that the flow-go repository should be cloned in the same folder as the DPS with its default name, so that the Go module replace statement works as intended: `replace github.com/onflow/flow-go/crypto => ./flow-go/crypto`.

* `git clone git@github.com:onflow/flow-go.git`
* `cd flow-go/crypto`
* `git checkout c0afa789365eb7a22713ed76b8de1e3efaf3a70a`
* `go generate`

You can then verify that the installation of the flow-go crypto package has been successful by running the tests of the project.

#### Build

You can build every binary by running `go build -tags="relic" -o . ./...` from the root of the repository.

### Setting up a test environment

In order to set up a test environment, it is recommended to use [Flow's integration tests](https://github.com/onflow/flow-go/tree/master/integration/localnet).

The first step is to install `flow-go` by following [this documentation](https://github.com/onflow/flow-go#installation) up until running `make install-tools`.

Then, you can head into the `integration/localnet` directory, and run `make init`. This will generate the necessary files to build and run nodes into a local Flow network.

If you want to generate smaller checkpoints and generate them quicker, you can edit the generated `docker-compose.nodes.yml` and add the following argument to the execution nodes: `--checkpoint-distance=1`.
Another recommended tweak is to edit the `SegmentSize` constant from `32 * 1024 * 1024` to simply `32 * 1024`. You can find this constant variable in `ledger/complete/wal/wal.go`.

Once you are happy with your configuration, you can run the local network by running `make start`.

Now, the local network is running, but nothing is happening since there are no transactions and accounts being registered on it.
You can then use [`flow-sim`](https://github.com/optakt/flow-sim) to create fake activity on your test network.
Simply clone the repository, run `go run main.go` and it should automatically start making transaction requests to your local network.

If you just need a valid checkpoint, you can monitor the state that your test network generates by running `watch ls data/consensus/<NodeID>` and waiting until you can see a file named `checkpoint.00000001` appear.

You can then copy part of this data folder to be used in DPS:

* `data/consensus/NodeID` can be given to the DPS as `data`
* `trie/execution/NodeID` can be given as `trie`
* `data/consensus/NodeID/checkpoint.00000001` can be given as `root.checkpoint`

You can then run the `flow-dps-indexer`, which should properly build its index based on the given information.

## More Resources

* [Flow Technical Papers](https://www.onflow.org/technical-paper)
* [Flow Developer Documentation](https://docs.onflow.org/)
* [Flow Developer Discord Server](https://onflow.org/discord)
* [Scalability Trilemma](https://vitalik.ca/general/2021/04/07/sharding.html)