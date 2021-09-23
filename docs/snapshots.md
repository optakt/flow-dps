# Index Snapshots

This document describes index snapshots, what they are and how they can be created or updated.

**Table of Contents**

- [What Are Index Snapshots](#what-are-index-snapshots)
- [Creating a Snapshot](#creating-a-snapshot)
- [Updating a Snapshot](#updating-a-snapshot)
    1. [New Index](#new-index)
    2. [New Snapshot](#new-snapshot)

## What Are Index Snapshots

Index snapshots are images of the DPS index at a certain point in time.
These images can be used to create or restore the content of an index in order to operate on a certain indexed data.
Index snapshots can be used to create a starting point for tests, so that tests have actual blocks, accounts, transactions and other information to operate on.

At a low level, snapshots are created using the [badger](https://github.com/dgraph-io/badger) backup functionality.
Technical documentation can be found [here](https://pkg.go.dev/github.com/dgraph-io/badger/v3#DB.Backup).

## Creating a Snapshot

Index snapshots are created using the CLI tool found at `cmd/create-index-snapshot`.
Usage for the tool is described in more detail [here](https://github.com/optakt/flow-dps/blob/master/cmd/create-index-snapshot/README.md).
By default, the snapshot is hex-encoded.

```console
$ create-index-snapshot -i <index_dir> > output.hex
```

When an index snapshot is created, it is compressed using a specific compression dictionary.
When restoring the index, the snapshot needs to be decompressed using the same dictionary or decompression will fail.

## Updating a Snapshot

It is sometimes necessary to regenerate the snapshot, for example when an index was added or changed, or because the internal format of the index changed (like when a compression dictionary is changed).

### New Index

The following command creates a new index.
Be careful to always use the same inputs as for the original index, if the goal is to continue building on the existing state.

```console
$ flow-dps-indexer -t <trie_dir> -d <data_dir> -i <index_dir> -a
```

### New Snapshot

Creating the new snapshot is the same as described in the [Creating a snapshot](#creating-a-snapshot) section, while referencing the new index.

```console
$ create-index-snapshot -i <new_index_dir> > output.hex
```
