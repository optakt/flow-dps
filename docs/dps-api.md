# DPS API Documentation

## Table of Contents

1. [Table of Contents](#table-of-contents)
2. [Endpoints](#endpoints)
3. [Types](#types)
    - [GetCommitRequest](#getcommitrequest)
    - [GetCommitResponse](#getcommitresponse)
    - [GetEventsRequest](#geteventsrequest)
    - [GetEventsResponse](#geteventsresponse)
    - [GetFirstRequest](#getfirstrequest)
    - [GetFirstResponse](#getfirstresponse)
    - [GetHeaderRequest](#getheaderrequest)
    - [GetHeaderResponse](#getheaderresponse)
    - [GetHeightRequest](#GetHeightRequest)
    - [GetHeightResponse](#GetHeightResponse)
    - [GetLastRequest](#getlastrequest)
    - [GetLastResponse](#getlastresponse)
    - [GetRegistersRequest](#getregistersrequest)
    - [GetRegistersResponse](#getregistersresponse)
    - [GetTransactionRequest](#GetTransactionRequest)
    - [GetTransactionResponse](#GetTransactionResponse)
    - [ListCollectionsForBlockRequest](#ListCollectionsForBlockRequest)
    - [ListCollectionsForBlockResponse](#ListCollectionsForBlockResponse)
    - [ListTransactionsForBlockRequest](#ListTransactionsForBlockRequest)
    - [ListTransactionsForBlockResponse](#ListTransactionsForBlockResponse)
    - [ListTransactionsForCollectionRequest](#ListTransactionsForCollectionRequest)
    - [ListTransactionsForCollectionResponse](#ListTransactionsForCollectionResponse)

## Endpoints

| Method Name                   | Request Type                                                                  | Response Type                                                                   |
|-------------------------------|-------------------------------------------------------------------------------|---------------------------------------------------------------------------------|
| GetCommit                     | [GetCommitRequest](#GetCommitRequest)                                         | [GetCommitResponse](#GetCommitResponse)                                         |
| GetEvents                     | [GetEventsRequest](#GetEventsRequest)                                         | [GetEventsResponse](#GetEventsResponse)                                         |
| GetFirst                      | [GetFirstRequest](#GetFirstRequest)                                           | [GetFirstResponse](#GetFirstResponse)                                           |
| GetHeader                     | [GetHeaderRequest](#GetHeaderRequest)                                         | [GetHeaderResponse](#GetHeaderResponse)                                         |
| GetHeight                     | [GetHeightRequest](#GetHeightRequest)                                         | [GetHeightResponse](#GetHeightResponse)                                         |
| GetLast                       | [GetLastRequest](#GetLastRequest)                                             | [GetLastResponse](#GetLastResponse)                                             |
| GetRegisters                  | [GetRegistersRequest](#GetRegistersRequest)                                   | [GetRegistersResponse](#GetRegistersResponse)                                   |
| GetTransaction                | [GetTransactionRequest](#GetTransactionRequest)                               | [GetTransactionResponse](#GetTransactionResponse)                               |
| ListCollectionsForBlock       | [ListCollectionsForBlockRequest](#ListCollectionsForBlockRequest)             | [ListCollectionsForBlockResponse](#ListCollectionsForBlockResponse)             |
| ListTransactionsForBlock      | [ListTransactionsForBlockRequest](#ListTransactionsForBlockRequest)           | [ListTransactionsForBlockResponse](#ListTransactionsForBlockResponse)           |
| ListTransactionsForCollection | [ListTransactionsForCollectionRequest](#ListTransactionsForCollectionRequest) | [ListTransactionsForCollectionResponse](#ListTransactionsForCollectionResponse) |

## Types

### GetCommitRequest

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

### GetCommitResponse

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |
| commit | `bytes`  |       |

### GetEventsRequest

| Field  | Type     | Label    |
|--------|----------|----------|
| height | `uint64` |          |
| types  | `string` | repeated |

### GetEventsResponse

| Field  | Type     | Label    |
|--------|----------|----------|
| height | `uint64` |          |
| types  | `string` | repeated |
| data   | `bytes`  |          |

The `data` field contains a [CBOR-encoded](https://cbor.io/) slice of Flow events (`[]flow.Event`) as payload.

Here is an example of how to decode this field in a small Go program:

```go
   var events []flow.Event
err := cbor.Unmarshal(response.Data, &events)
if err != nil {
return err
}
```

### GetFirstRequest

For now, `GetFirstRequest` is empty.

### GetFirstResponse

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

### GetHeaderRequest

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

### GetHeaderResponse

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |
| data   | `bytes`  |       |

The `data` field contains a [CBOR-encoded](https://cbor.io/) Flow header (`flow.Header`) as payload.

Here is an example of how to decode this field in a small Go program:

```go
   var header flow.Header
err := cbor.Unmarshal(response.Data, &header)
if err != nil {
return err
}
```

### GetHeightRequest

| Field   | Type    | Label |
|---------|---------|-------|
| blockID | `bytes` |       |

### GetHeightResponse

| Field   | Type     | Label |
|---------|----------|-------|
| blockID | `bytes`  |       |
| height  | `uint64` |       |

### GetLastRequest

For now, `GetLastRequest` is empty.

### GetLastResponse

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

### GetRegistersRequest

| Field  | Type     | Label    |
|--------|----------|----------|
| height | `uint64` |          |
| paths  | `bytes`  | repeated |

### GetRegistersResponse

| Field  | Type     | Label    |
|--------|----------|----------|
| height | `uint64` |          |
| paths  | `bytes`  | repeated |
| values | `bytes`  | repeated |

### GetTransactionRequest

| Field         | Type    | Label |
|---------------|---------|-------|
| transactionID | `bytes` |       |

### GetTransactionResponse

| Field         | Type    | Label |
|---------------|---------|-------|
| transactionID | `bytes` |       |
| data          | `bytes` |       |

### ListCollectionsForBlockRequest

| Field   | Type    | Label |
|---------|---------|-------|
| blockID | `bytes` |       |

### ListCollectionsForBlockResponse

| Field         | Type    | Label    |
|---------------|---------|----------|
| blockID       | `bytes` |          |
| collectionIDs | `bytes` | repeated |

### ListTransactionsForBlockRequest

| Field   | Type    | Label |
|---------|---------|-------|
| blockID | `bytes` |       |

### ListTransactionsForBlockResponse

| Field          | Type    | Label    |
|----------------|---------|----------|
| blockID        | `bytes` |          |
| transactionIDs | `bytes` | repeated |

### ListTransactionsForCollectionRequest

| Field        | Type    | Label |
|--------------|---------|-------|
| collectionID | `bytes` |       |

### ListTransactionsForCollectionResponse

| Field          | Type    | Label    |
|----------------|---------|----------|
| collectionID   | `bytes` |          |
| transactionIDs | `bytes` | repeated |