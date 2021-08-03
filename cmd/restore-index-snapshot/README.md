# Restore Index Snapshot

## Description

This utility binary restores indexes from previously created snapshots.
The `input` argument for the utility should point to the `gzip` archive created by the `create-index-snapshot` tool.
This utility will create a new index, and use the badger API to restore the database.
It also logs any metadata written by the backup tool (such as the time of backup).

## Usage

```sh
Usage of ./restore-index-snapshot:
  -i, --index string   database directory for state index (default "index")
      --input string   snapshot archive path
  -l, --level string   log output level (default "info")
```

## Example

Restore the index to `new_index` from a file `flow-dps-snapshot-03-08-2021-11-02.gz`:

```sh
restore-index-snapshot -i new_index --input ./flow-dps-snapshot-03-08-2021-11-02.gz 2> >(jq)
{
  "level": "info",
  "file": "./flow-dps-snapshot-03-08-2021-11-02.gz",
  "time": "2021-08-03T09:45:00Z",
  "message": "snapshot archive open ok"
}
{
  "level": "info",
  "comment": "DPS Index snapshot created at 03-08-2021-09-02",
  "time": "2021-08-03T09:45:00Z",
  "message": "snapshot archive info"
}
{
  "level": "info",
  "archive_time": "2021-08-03T11:02:04+02:00",
  "time": "2021-08-03T09:45:00Z",
  "message": "snapshot archive creation time"
}
{
  "level": "info",
  "time": "2021-08-03T09:45:00Z",
  "message": "snapshot restore complete"
}
```
