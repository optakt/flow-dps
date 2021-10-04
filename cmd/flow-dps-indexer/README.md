# Flow DPS Indexer

## Description

The Flow DPS Indexer binary implements the core functionality to create the index for past sporks.
It needs a reference to the protocol state database of the spork, as well as the trie directory and an execution state checkpoint.
The index is generated in the form of a Badger database that allows random access to any ledger register at any block height.

## Usage

```sh
Usage of flow-dps-indexer:
  -c, --checkpoint string   path to root checkpoint file for execution state trie
  -d, --data string         path to database directory for protocol data (default "data")
  -f, --force               force indexing to bootstrap from root checkpoint and overwrite existing index
  -i, --index string        path to database directory for state index (default "index")
  -l, --level string        log output level (default "info")
  -s, --skip                skip indexing of execution state ledger registers
  -t, --trie string         path to data directory for execution state ledger
```

## Example

The below command line starts indexing a past spork from the on-disk information.

```sh
./flow-dps-indexer -a -l debug -d /var/flow/data/protocol -t /var/flow/data/execution -c /var/flow/bootstrap/root.checkpoint -i /var/flow/data/index
```
