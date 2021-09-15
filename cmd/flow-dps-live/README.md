# Flow DPS Live

## Description

The Flow DPS Live binary implements the core functionality to create the index for live sporks.
It needs access to a Google Cloud Storage bucket containing the execution state in the form of block data files, as well as access to the Flow network as an unstaked consensus follower.
The index is generated in the form of a Badger database that allows random access to any ledger register at any block height.

## Usage

```sh
Usage of flow-dps-live:
  -a, --address string        address to serve the GRPC DPS API on (default "127.0.0.1:5005")
  -b, --bootstrap string      path to directory with public bootstrap information for the spork
  -u, --bucket string         name of the Google Cloud Storage bucket which contains the block data
  -c, --checkpoint string     checkpoint file for state trie
  -d, --data string           database directory for protocol data
  -f, --force                 overwrite existing index database
  -i, --index string          database directory for state index (default "index")
  -l, --level string          log output level (default "info")
  -s, --skip                  skip indexing of execution state ledger registers
      --seed-address string   address of the seed node to follow unstaked consensus
      --seed-key string       hex-encoded public network key of the seed node to follow unstaked consensus
```

## Example

The below command line starts indexing a live spork.

```sh
./flow-dps-live -u flow-block-data -i /var/flow/index -d /var/flow/data -c /var/flow/bootstrap/root.checkpoint -b /var/flow/bootstrap/public --seed-address access.canary.nodes.onflow.org:9000 --seed-key cfce845fa9b0fb38402640f997233546b10fec3f910bf866c43a0db58ab6a1e4
```
