# Flow Rosetta Server

## Description

The Flow Rosetta Server implements the Rosetta Data API specifications for the Flow network.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.
Flow core contract addresses are derived from the chain ID with which the service is started.
This allows the Rosetta API to access state remotely, or locally by running the Flow DPS Server on the same host.

## Usage

```sh
Usage of flow-rosetta-server:
  -a, --api string              host URL for GRPC API endpoint (default "127.0.0.1:5005")
  -e, --cache uint              maximum cache size for register reads in bytes (default 1073741824)
  -l, --level string            log output level (default "info")
  -p, --port uint16             port to host Rosetta API on (default 8080)
  -t, --transaction-limit int   maximum amount of transactions to include in a block response (default 200)
```

## Example

The following command line starts the Flow DPS Rosetta server for a main network spork on port 8080.
It uses a local instance of the Flow DPS Server for access to the execution state.

```sh
./flow-dps-rosetta -a "127.0.0.1:5005" -p 8080
```
