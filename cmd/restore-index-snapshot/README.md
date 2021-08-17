# Restore Index Snapshot

## Description

This utility binary restores indexes from previously created snapshots.
Util reads data from `stdin`, which should be the snapshot data created by the `create-index-snapshot` tool.
The `format` argument specifies which snapshot format was used - hex, raw or gzip.
This utility will create a new index, and use the badger API to restore the database.
It also logs any metadata written by the backup tool (such as the time of backup).

## Usage

```sh
Usage of ./restore-index-snapshot:
  -c, --compression string   compression algorithm ("none", "zstd" or "gzip") (default "zstd")
  -e, --encoding string      output encoding ("none", "hex" or "base64") (default "none")
  -i, --index string         database directory for state index (default "index")
```

## Example

Restore the index to `new_index` from a file `flow-dps-snapshot-03-08-2021-11-02.gz`:

```sh
cat flow-dps-snapshot-03-08-2021-11-02.gz | restore-index-snapshot --format gzip -i new-index 2> >(jq)
{
  "level": "info",
  "comment": "DPS Index snapshot created at 16-08-2021-08-56",
  "time": "2021-08-16T08:56:48Z",
  "message": "snapshot archive info"
}
{
  "level": "info",
  "archive_time": "2021-08-16T10:56:31+02:00",
  "time": "2021-08-16T08:56:48Z",
  "message": "snapshot archive creation time"
}
{
  "level": "info",
  "time": "2021-08-16T08:56:48Z",
  "message": "snapshot restore complete"
}
```
