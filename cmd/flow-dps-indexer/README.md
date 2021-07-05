# Flow DPS Indexer

## Description

The Flow DPS Indexer binary implements the core functionality to create the index for past sporks.
It needs a reference to the protocol state database of the spork, as well as the trie directory and an execution state checkpoint.
The index is generated in the form of a Badger database that allows random access to any ledger register at any block height.

## Usage

```sh
Usage of flow-dps-indexer:
  -c, --checkpoint string    checkpoint file for state trie
  -d, --data string          database directory for protocol data
  -f, --force                overwrite existing index database
  -i, --index string         database directory for state index (default "index")
  -a, --index-all            index everything
  -o, --index-collections    index collections
  -m, --index-commits        index commits
  -e, --index-events         index events
      --index-guarantees     index collection guarantees
  -h, --index-headers        index headers
  -p, --index-payloads       index payloads
      --index-results        index transaction results
      --index-seals          index seals
  -x, --index-transactions   index transactions
  -l, --level string         log output level (default "info")
  -m, --metrics                     enable metrics collection and output
      --metrics-interval duration   defines the interval of metrics output to log (default 5m0s)
      --skip-bootstrap              enable skipping checkpoint register payloads indexing
  -t, --trie string          data directory for state ledger
```

## Example

The below command line starts indexing a past spork from the on-disk information.

```sh
./flow-dps-indexer -a -l debug -d /var/flow/data/protocol -t /var/flow/data/execution -c /var/flow/bootstrap/root.checkpoint -i /var/flow/data/index
```
