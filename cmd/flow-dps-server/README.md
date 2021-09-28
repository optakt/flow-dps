# Flow DPS Server

## Description

The Flow DPS Server runs on top of a DPS index to provide random access to the execution state through its API.
Both the Flow DPS Indexer and the Flow DPS Live tool can create such an index.
In the case of the indexer, the index is static and built from a previous spork's state.
For the live tool, the index is dynamic and updated on an ongoing basis from the data sent from a Flow execution node.
Access to the execution state is provided through a GRPC API.

## Usage

```sh
Usage of flow-dps-server:
  -a, --address string  bind address for serving DPS API (default "127.0.0.1:5005")
  -i, --index string    path to database directory for state index (default "index")
  -l, --log string      log output level (default "info")
```

## Example

The following command line starts the DPS GRPC API server to serve requests at the address "172.17.0.1:5005".

```sh
./flow-dps-server -i /var/flow/data/index -a 172.17.0.1:5005
```
