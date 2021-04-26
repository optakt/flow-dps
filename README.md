# Flow Data Provisioning Service

## Control Flow

1. Open protocol state and use state commitment from root seal as initial checkpoint.
2. Open LedgerWAL and stream updates into Trie until checkpoint state commitment is reached.
3. Merge and de-duplicate all register updates from pending trie updates up to checkpoint and store as delta that results in sealed state commitment after block with corresponding height and hash is sealed.
4. Continue doing until we have a mapping of one delta for each block (might be empty).
5. Step backwards from last sealed block to root block and index each register value upon change.

=> We need the height and block ID of each seal, on top of the state commitment.