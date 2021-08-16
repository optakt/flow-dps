# Create Index Snapshot

## Description

This utility binary creates snapshots of existing indexes.
When a path to the index (badger database) is specified, the badger API is used to create a backup. 
This backup is written to the standard output, using the format specified by the `format` argument.
This backup is compressed with Zstandard compression.

This output can be used to restore a database from a previous snapshot by a `restore-index-snapshot` tool.

## Usage

```sh
Usage of create-index-snapshot:
  -f, --format string   output format (hex, gzip or raw) (default "raw")
  -i, --index string    database directory for state index (default "index")
  -l, --level string    log output level (default "info")
```

## Examples

### Usage

Backup index to a hex-encoded string:

```console
$ create-index-snapshot -i index --format hex > snapshot.hex
```

Backup index to a file named `dps-backup.gz` in the `/tmp` directory:

```console
$ create-index-snapshot -i index --format gzip > /tmp/dps-backup.gz
```


### Go Program Restoring the Index

The program below opens a read-only in-memory badger database and restores the state from the created hex-encoded backup. Error handling is omitted for brevity.

```go
opts := badger.DefaultOptions("").WithInMemory(true).WithReadOnly(true).WithLogger(nil)
db, _ := badger.Open(opts)

payload := "hex output of create-index-snapshot"

dbSnapshot, _ := zstd.NewReader(hex.NewDecoder(strings.NewReader(payload)))
defer dbSnapshot.Close()

db.Load(dbSnapshot, 10)
```
