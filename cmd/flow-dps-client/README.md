# Flow DPS Client

## Description

The Flow DPS Client provides access to a Flow DPS Server's index through the command line. 
It can be used to execute Cadence scripts at an arbitrary block height of a fork.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.

## Usage

```sh
Usage of flow-dps-client:
  -a, --api string      host for GRPC API server
  -e, --cache uint      maximum cache size for register reads in bytes (default 1000000000)
  -h, --height uint     block height to execute the script at
  -l, --level string    log output level (default "info")
  -p, --params string   comma-separated list of Cadence parameters
  -s, --script string   path to file with Cadence script (default "script.cdc")
```

Cadence parameters can be provided as a list of comma-separated `Type(Value)` pairs.
Whenever raw bytes are represented, they should be given in hexadecimal format.

`-p "UFix64(123.456),String(/storage/FlowTokenVault),Bytes(43F164656E636521467572AC76657)"`.

## Example

The following executes a Cadence script by using state retrieved from the given GRPC API.

```sh
./flow-dps-client -a "127.0.0.1:5005" -s "get_balance.cdc" -p "Address(436164656E636521)"
```
