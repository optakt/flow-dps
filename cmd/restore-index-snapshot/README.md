# Restore Index Snapshot

## Description

This utility binary restores snapshots of DPS state index databases.
It uses the Badger backup API to load a single file snapshot of the database.
Input is read from the standard input and a file can be piped into the binary if desired.
The user must indicate which encoding and compression formats were used during snapshot creation.

A new index database will be created at the indicated directory.
The restoration will fail if an DPS index database already exists at the given path.

## Usage

```sh
Usage of ./restore-index-snapshot:
  -c, --compression string   compression algorithm ("none", "zstd" or "gzip") (default "zstd")
  -e, --encoding string      output encoding ("none", "hex" or "base64") (default "none")
  -i, --index string         database directory for state index (default "index")
```

## Example

Restore a DPS index database from a Gzip compressed file without encoding:

```console
$ restore-index-snapshot -i /var/dps/index -c gzip < dps-index-snapshot.gz
```
