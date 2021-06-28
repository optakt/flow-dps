# Index snapshots

This document describes index snapshots, what they are and how they can be created or updated.

**Table of Contents**
- [What are index snapshots](#what-are-snapshots)
- [Create a snapshot](#create-a-snapshot)
- [Updating a snapshot](#update-a-snapshot)
    1. [Create a new index](#create-a-new-index)
    2. [Create an index snapshot](#create-an-index-snapshot)

## What are index snapshots

Index snapshots are images of the DPS index at a certain point in time.
These images can be used to create or restore the content of an index in order to operate on a certain indexed data.
Index snapshots can be used to create a starting point for tests, so that tests have actual blocks, accounts, transactions and other information to operate on.

At a low level, snapshots are created using the [badger](https://github.com/dgraph-io/badger) backup functionality.
Technical documentation can be found [here](https://pkg.go.dev/github.com/dgraph-io/badger/v3#DB.Backup).

## Create a snapshot

Index snapshots are created using the CLI tool found at `cmd/create-index-snapshot`.
Usage for the tool is described in more detail [here](https://github.com/optakt/flow-dps/blob/master/cmd/create-index-snapshot/README.md).

## Update a snapshot

It is sometimes necessary to regenerate the snapshot.
Perhaps in order to add additional blocks, events or some other data to the snapshot to be referenced in tests.

Other reasons may be because the internal format of the index changed, like when a compression dictionary is updated.
When an index snapshot is created, it is compressed using a specific compression dictionary.
When restoring the index, the snapshot needs to be decompressed using the same dictionary or decompression will fail.

### Create a new index

Create a new index using the same inputs as for the original index.

```console
$ flow-dps-indexer -t <trie_dir> -d <data_dir> -i <index_name> -a
```

### Create an index snapshot

Creating an index snapshot is the same as described in the [Create a snapshot](#create-a-snapshot) section, while referencing the new index.

```console
$ create-index-snapshot -i <path_to_new_index> > output.hex
```

By default the snapshot is hex-encoded.