# Extract Ledger Payloads

## Description

This utility binary can be used to extract a randomized set of payloads from a state trie.
It must be run on top of an execution state, with access to the corresponding protocol state.
Upon start, it will reconstruct the state trie from the checkpoint and the write-ahead log.
Once the final state trie has been reached, it will extract a randomized set of payloads.
The payloads will be written to one file each, until the configured total size is reached.

This is useful for the creation of dictionaries that can be used with Zstandard compression.
The default size for a Zstandard dictionary is set to 112,640 bytes, or ~112 kilobytes.
The training command expects a set of compressable payloads as an input, with one payload per file.
It is recommended to use at least ten times the size of the dictionary for total input size.
Ideally, a hundred times the size of the dictionary is best, which is what we use as default here.

## Usage

```sh
Usage: extract-ledger-payloads [options]

Options:
  -l, --log-level       log level for JSON logger output (default: info)
  -d, --data-dir        directory for protocol state database (default: data)
  -t, --trie-dir        directory for execution state database (default: trie)
  -c, --checkpoint      file containing state trie snapshot (default: root.checkpoint)
  -o, --output-dir      directory for output of ledger payloads (default: payloads)
  -s, --size-limit      limit for total size of output files (default: 11264000) 
```

## Example

```sh
./extract-ledger-payloads -l debug -d /var/flow/data -t /var/flow/trie -c /var/flow/trie/root.checkpoint -o ./payloads
```