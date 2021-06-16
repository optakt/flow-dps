# Flow DPS Client

## Description

The Flow DPS Client provides access to a Flow DPS Server's index through the command line. 
It can be used to execute Cadence scripts at an arbitrary block height of a fork.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.

## Usage

```sh
Usage of flow-dps-client:
  -a, --api string      host for GRPC API server (default "127.0.0.1:5005")
  -h, --height uint     block height to execute the script at
  -l, --log string      log output level (default "info")
  -p, --params string   comma-separated list of Cadence parameters
  -s, --script string   path to Cadence script file (default "script.cdc")
```

Cadence parameters can be provided as a list of comma-separated `Type(Value)` pairs.
Whenever raw bytes are represented,they should be given in hexadecimal format.

Example: `-p "UFix64(123.456),Address(436164656E636521),Bytes(436164656E6365214675726576657)"`.

## Example

The following executes a Cadence script by using state retrieved from the given GRPC API.

```sh
./flow-dps-client -a "127.0.0.1:5005" -s "get_balance.cdc" -p "String(Flow),UFix64(10.0)"
```