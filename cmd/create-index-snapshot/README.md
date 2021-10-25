# Create Index Snapshot

## Description

This utility binary creates snapshots of DPS state index databases.
It uses the Badger backup API to create a single file snapshot of the database.
Output is written to standard output and can be piped into a file if desired.
The user can choose between various encoding and compression formats.

The index database can later be restored using the `restore-index-snapshot` tool.

## Usage

```sh
Usage of create-index-snapshot:
  -c, --compression string   compression algorithm ("none", "zstd" or "gzip") (default "zstd")
  -e, --encoding string      output encoding ("none", "hex" or "base64") (default "none")
  -i, --index string         database directory for state index (default "index")
      --readonly             open database as read-only (default true)
```

## Examples

### Usage

Outputting a hex-encoded zstd-compressed index snapshot on the console to use in testing:

```console
$ create-index-snapshot -i /var/dps/index -c zstd -e hex
```

Back up an existing index database to a Gzip compressed file without encoding:

```console
$ create-index-snapshot -i /var/dps/index -c gzip > dps-index-snapshot.gz
```

### Go Program Restoring the Index

The program below opens a in-memory Badger database and restores the state from the created hex-encoded backup. Error handling is omitted for brevity.

```go
opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
db, _ := badger.Open(opts)

payload := "<pasted hex-encoded zstd-compressed output of create-index-snapshot>"

reader, _ := zstd.NewReader(hex.NewDecoder(strings.NewReader(payload)))
defer reader.Close()

db.Load(reader, 10)
```
