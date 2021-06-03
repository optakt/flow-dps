# Flow DPS Client

## Description

The Flow DPS Client provides access to a Flow DPS Server's index throught the command line. 
It can be used to execute Cadence scripts at an arbitrary block height of a fork.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.

## Usage

```sh
Usage of flow-dps-client:
  -a, --api string      host for GRPC API server (default "127.0.0.1:5005")
  -h, --height uint     block height to execute the script at
  -l, --log string      log output level (default "info")
  -p, --params string   JSON encoded Cadence parameters for script execution
  -s, --script string   path to Cadence script file (default "script.cdc")
```

## Example

The following executes a Cadence script by using state retrieved from the given GRPC API.

```sh
./flow-dps-client -a "127.0.0.1:5005" -s "get_balance.cdc"
```