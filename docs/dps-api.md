# DPS API Documentation

## Table of Contents

- [Endpoints](#Endpoints)
- [Types](#Types)
    - [GetCommitRequest](#GetCommitRequest)
    - [GetCommitResponse](#GetCommitResponse)
    - [GetEventsRequest](#GetEventsRequest)
    - [GetEventsResponse](#GetEventsResponse)
    - [GetHeaderRequest](#GetHeaderRequest)
    - [GetHeaderResponse](#GetHeaderResponse)
    - [GetLastRequest](#GetLastRequest)
    - [GetLastResponse](#GetLastResponse)
    - [GetRegistersRequest](#GetRegistersRequest)
    - [GetRegistersResponse](#GetRegistersResponse)

## Endpoints

| Method Name  | Request Type                                | Response Type                                 |
|--------------|---------------------------------------------|-----------------------------------------------|
| GetLast      | [GetLastRequest](#GetLastRequest)           | [GetLastResponse](#GetLastResponse)           |
| GetHeader    | [GetHeaderRequest](#GetHeaderRequest)       | [GetHeaderResponse](#GetHeaderResponse)       |
| GetCommit    | [GetCommitRequest](#GetCommitRequest)       | [GetCommitResponse](#GetCommitResponse)       |
| GetEvents    | [GetEventsRequest](#GetEventsRequest)       | [GetEventsResponse](#GetEventsResponse)       |
| GetRegisters | [GetRegistersRequest](#GetRegistersRequest) | [GetRegistersResponse](#GetRegistersResponse) |

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

The `data` field contains a [CBOR-encoded](https://cbor.io/) slice of Flow events (`[]flow.Event`) as payload.

Here is an example of how to decode this field in a small Go program:

```go
   var events []flow.Event
   err := cbor.Unmarshal(response.Data, &events)
   if err != nil {
     return err
   }
```

| Field  | Type     | Label    |
|--------|----------|----------|
| height | `uint64` |          |
| types  | `string` | repeated |
| data   | `bytes`  |          |

### GetHeaderRequest

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

### GetHeaderResponse

The `data` field contains a [CBOR-encoded](https://cbor.io/) Flow header (`flow.Header`) as payload.

Here is an example of how to decode this field in a small Go program:

```go
   var header flow.Header
   err := cbor.Unmarshal(response.Data, &header)
   if err != nil {
     return err
   }
```

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |
| data   | `bytes`  |       |

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