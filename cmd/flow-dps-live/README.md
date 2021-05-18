# Flow DPS Live

## Description

The Flow DPS Live binary implements the core functionality to create the index for live sporks.
It needs the address and ports to access an execution node's pub and req sockets, and an execution state checkpoint.
The index is generated in the form of a Badger database that allows random access to any ledger register at any block height.

## Usage

```sh
Usage of flow-dps-live:
  -n, --node-host           hostname or IP of the live node
  -p, --pub-port            port on which the pub socket is exposed (default 14532)
  -r, --req-port            port on which the req socket is exposed (default 14533)
  -c, --checkpoint string   checkpoint file for state trie
  -i, --index string        database directory for state index (default "index")
  -l, --log string          log output level (default "info")
```

## Example

The following command line starts indexing a live spork from the socket.

```sh
./flow-dps-live -n 172.17.100.42 -c /var/flow/bootstrap/root.checkpoint -i /var/flow/data/index
```