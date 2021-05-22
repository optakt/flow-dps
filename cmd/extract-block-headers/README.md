# Extract Block Headers

## Description

This utility binary can be used to extract a randomized set of block headers from a protocol state.
Upon start, it will extract random headers within the provided height range from the database.
The headers will be written to one file each, until the configured total size is reached.

This is useful for the creation of dictionaries that can be used with Zstandard compression.
The default size for a Zstandard dictionary is set to 112,640 bytes, or ~112 kilobytes.
The training command expects a set of compressable payloads as an input, with one payload per file.
It is recommended to use at least ten times the size of the dictionary for total input size.
Ideally, a hundred times the size of the dictionary is best, which is what we use as default here.

## Usage

```sh
Usage: extract-block-headers [options]

Options:
  -l, --log-level       log level for JSON logger output (default: info)
  -d, --data-dir        directory for protocol state database (default: data)
  -b, --begin-height    lowest block height to include in extraction (default: 0)
  -f, --finish-height   highest block height to include in extraction (default: 100000000)
  -o, --output-dir      directory for output of ledger payloads (default: headers)
  -s, --size-limit      limit for total size of output files (default: 11264000) 
```

## Example

```sh
./extract-block-headers -l debug -d /var/flow/data -b 1200000 -f 1500000 -o ./headers
```