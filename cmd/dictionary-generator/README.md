# Dictionary Generator

## Description

This utility binary generates [Zstandard compression dictionaries](http://facebook.github.io/zstd/#small-data) for
ledger payloads, events and transactions. It does so by generating multiple dictionaries and incrementing their size
progressively, benchmarking them to compare them, and stops when doubling the size of the dictionaries leads to
negligible improvements in compression ratios. It then automatically transforms those dictionaries into Go files,
ready to be used by the `codec/zbor` package.

## Usage

```sh
Usage of dictionary-generator:
    -i, --index string         path to database directory for state index (default "index")
    -l, --level string         log output level (default "info")
    --dictionary-path string   path to the package in which to write dictionaries (default "./codec/zbor")
    --sample-path string       path to the directory in which to create temporary samples for dictionary training (default "./samples")
    --sample-size int          size of the sample dataset used for benchmarking (higher values increase accuracy at the expense of speed) (default 16384)
    --start-size int           minimum dictionary size to generate (will be doubled on each iteration) (default 512)
    --tolerance float          compression ratio increase tolerance (between 0 and 1) (default 0.1)
```

## Example

The below command line generates dictionaries in the path ./package/test/ and benchmarks the dictionaries with 100kb of samples.

```sh
./dictionary-generator --dictionary-path ./package/test --sample-size 102400
```
