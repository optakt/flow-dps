# Create Index Snapshot

## Description

This utility binary can be used to create a snapshot of an existing index. When a path to the index (badger database) is specified, the badger API is used to create a backup. This backup is compressed with Zstandard compression, encoded to hex and printed on standard output.

This output can be used to restore a database from a previous snapshot.

## Usage

```sh
Usage of create-index-snapshot:
  -d, --dir string   path to badger database
  -l, --log string   log level for JSON logger (default "info")
  -r, --raw string   target file for raw output (overwrites existing)
```

## Examples

```console
$ ./create-index-snapshot -d /path/to/index                   # write snapshot to stdout (hex)
$ ./create-index-snapshot -d /path/to/index --raw file.bin    # write snapshot to file (binary)
```

## Example

The program below opens a read-only in-memory badger database and restores the state from the created backup. Error handling is omitted for brevity.

```go
opts := badger.DefaultOptions("").WithInMemory(true).WithReadOnly(true).WithLogger(nil)
db, _ := badger.Open(opts)

payload := "output of create-index-snapshot"

dbSnapshot, _ := zstd.NewReader(hex.NewDecoder(strings.NewReader(payload)))
defer dbSnapshot.Close()

db.Load(dbSnapshot, 10)
```