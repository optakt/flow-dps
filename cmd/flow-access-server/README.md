# Flow Access Server

## Description

The Flow Access Server runs on top of a DPS index to implement the [Flow Access API](https://docs.onflow.org/access-api).
Both the Flow DPS Indexer and the Flow DPS Live tool can create such an index.
In the case of the indexer, the index is static and built from a previous spork's state.
For the live tool, the index is dynamic and updated on an ongoing basis from the data sent from a Flow execution node.

## Usage

```sh
Usage of flow-access-server:
  -i, --index string   database directory for state index (default "index")
  -l, --log string     log output level (default "info")
  -p, --port uint16    port to serve GRPC API on (default 5006)
```

## Example

The following command line starts the DPS Access API server to serve requests on port `5006`.

```sh
./flow-access-server -i /var/flow/data/index -p 5006
```
