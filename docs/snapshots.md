# Index Snapshots

This document describes index snapshots, what they are and how they can be created and restored.

**Table of Contents**

- [What Are Index Snapshots](#what-are-index-snapshots)
- [Creating a Snapshot](#creating-a-snapshot)
- [Restoring a Snapshot](#restoring-a-snapshot)

## What Are Index Snapshots

Index snapshots are images of the DPS index database.
These images can be used to easily transfer DPS index snapshots as a single file or to archive them in a more space-efficient manner.
They can also be used in testing, so that tests have actual blocks, accounts, transactions and other information to operate on.

At a low level, snapshots are created using the [badger](https://github.com/dgraph-io/badger) backup functionality.
Technical documentation can be found [here](https://pkg.go.dev/github.com/dgraph-io/badger/v2#DB.Backup).

## Creating a Snapshot

Index snapshots are created using the CLI tool found at `cmd/create-index-snapshot`.
Usage for the tool is described in more detail [here](https://github.com/optakt/flow-dps/blob/master/cmd/create-index-snapshot/README.md).
By default, the snapshot is not encoded and output contains raw (binary) data.

```console
$ create-index-snapshot -i <index_dir> > output.bin
```

When an index snapshot is created, it can be compressed with a specific compression algorithm (zstd or gzip).
When restoring the index, the snapshot needs to be decompressed using the same algorithm or snapshot restore will fail.

## Restoring a Snapshot

To restore a snapshot, use the CLI tool found at `cmd/restore-index-snapshot`.
Usage for the tool is described in more detail [here](https://github.com/optakt/flow-dps/blob/master/cmd/restore-index-snapshot/README.md).
To successfully restore the snapshot, you must specify the same compression and encoding options as used when creating it.

Example of restoring a gzip compressed snapshot:
```console
$ restore-index-snapshot -i /var/dps/index -c gzip < dps-index-snapshot.gz
```
