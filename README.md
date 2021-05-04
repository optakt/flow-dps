# Flow Data Provisioning Service

## Road Map

- bootstrap from checkpoint
- add proper logging

- index transaction events
- events retrieval interface
- clean up component interfaces
- build snapshot creator
- REST API interface
- DPS web client

## Interfaces

1. Indexer: indexes changes/deltas per block
2. Update Stream: gets all the trie updates per chunk
3. Chain View: has block sequence data
