# Flow DPS Client

## Description

The Flow DPS Client allows access to a Flow DPS Server's index throught the command line. 
It can be used to execute a Cadence script at an arbitrary block height of a fork.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.

## Usage

```sh
Usage of flow-dps-client:
  -a, --api string     host URL for GRPC API endpoint (default "127.0.0.1:5005")
  -l, --log string     log output level (default "info")
  -s, --script string  path to the Cadence script file to be executed (default "script.cdc")
```

## Example

The below command line starts the Flow DPS Rosetta server for a main network spork on port 8080.
It uses a local instance of the Flow DPS Server for access to the execution state.

```sh
./flow-dps-rosetta -a "127.0.0.1:5005" -c "flow-mainnet" -p 8080
```