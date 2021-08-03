# Flow DPS Live

## Description

The Flow DPS Live binary implements the core functionality to create the index for live sporks.
It needs access to an S3 bucket containing the execution state in the form of ledger WAL checkpoints, as well as access to the Flow network as a follower.
The index is generated in the form of a Badger database that allows random access to any ledger register at any block height.

## Usage

```sh
Usage of flow-dps-live:
      --access-address string       address (host:port) of the peer to connect to
      --access-key string           network public key of the peer to connect to
      --bind-addr string            address on which to bind the FIXME (default "127.0.0.1:FIXME")
  -b, --bucket string               name of the S3 bucket which contains the state ledger
  -c, --checkpoint string           checkpoint file for state trie
  -d, --data string                 database directory for protocol data
      --download-directory string   directory where to download ledger WAL checkpoints
  -f, --force                       overwrite existing index database
  -i, --index string                database directory for state index (default "index")
  -a, --index-all                   index everything
      --index-collections           index collections
      --index-commits               index commits
      --index-events                index events
      --index-guarantees            index collection guarantees
      --index-headers               index headers
      --index-payloads              index payloads
      --index-results               index transaction results
      --index-seals                 index seals
      --index-transactions          index transactions
  -l, --level string                log output level (default "info")
  -m, --metrics                     enable metrics collection and output
      --metrics-interval duration   defines the interval of metrics output to log (default 5m0s)
  -n, --node-id string              node id to use for the DPS
  -p, --port uint16                 port to serve GRPC API on (default 5005)
      --skip-bootstrap              enable skipping checkpoint register payloads indexing
```

## Example

The below command line starts indexing a live spork.

```sh
./flow-dps-live -a -l debug -b myS3BucketName -r us-west-2 -c /var/flow/bootstrap/root.checkpoint -i /var/flow/data/index
```
