# Extract Block Events

## Description

This utility binary can be used to extract a randomized set of transaction events from a protocol state.
Upon start, it will extract the events for a random height within the provided height range from the database.
It will then extract all events of a randomly chosen event type from the event types present in the batch.
The batches will be written to one file each, until the configured total size is reached.

This is useful for the creation of dictionaries that can be used with Zstandard compression.
The default size for a Zstandard dictionary is set to 112,640 bytes, or ~112 kilobytes.
The training command expects a set of compressable payloads as an input, with one payload per file.
It is recommended to use at least ten times the size of the dictionary for total input size.
Ideally, a hundred times the size of the dictionary is best, which is what we use as default here.

## Usage

```sh
Usage: extract-block-events [options]

Options:
  -l, --log-level       log level for JSON logger output (default: info)
  -d, --data-dir        directory for protocol state database (default: data)
  -b, --begin-height    lowest block height to include in extraction (default: 0)
  -f, --finish-height   highest block height to include in extraction (default: 100000000)
  -o, --output-dir      directory for output of ledger payloads (default: events)
  -s, --size-limit      limit for total size of output files (default: 11264000)
  -g, --group-size      maximum number of events to extract per block (default: 10)
```

## Example

```sh
./extract-block-events -l debug -d /var/flow/data -b 1200000 -f 1500000 -o ./events
```