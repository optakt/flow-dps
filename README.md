# Flow Data Provisioning Service

## Road Map

- bootstrap from checkpoint
  - (is the first update the root checkpoint?)
    - check Flow Go code
- add proper logging
- clean up component interfaces

- index transaction events
- events retrieval interface
- REST API interface
- DPS web client

## Interfaces

1. Indexer: indexes changes/deltas per block
2. Update Stream: gets all the trie updates per chunk
3. Chain View: has block sequence data