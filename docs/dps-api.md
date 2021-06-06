# DPS API Documentation

## Table of Contents

1. [Table of Contents](#table-of-contents)
2. [Endpoints](#endpoints)
3. [Types](#types)
   1. [GetFirstRequest](#getfirstrequest)
   2. [GetFirstResponse](#getfirstresponse)
   3. [GetLastRequest](#getlastrequest)
   4. [GetLastResponse](#getlastresponse)
   5. [GetHeaderRequest](#getheaderrequest)
   6. [GetHeaderResponse](#getheaderresponse)
   7. [GetCommitRequest](#getcommitrequest)
   8. [GetCommitResponse](#getcommitresponse)
   9. [GetEventsRequest](#geteventsrequest)
   10. [GetEventsResponse](#geteventsresponse)
   11. [GetRegistersRequest](#getregistersrequest)
   12. [GetRegistersResponse](#getregistersresponse)

## Endpoints

| Method Name  | Request Type                                | Response Type                                 |
|--------------|---------------------------------------------|-----------------------------------------------|
| GetFirst     | [GetFirstRequest](#GetFirstRequest)         | [GetFirstResponse](#GetFirstResponse)         |
| GetLast      | [GetLastRequest](#GetLastRequest)           | [GetLastResponse](#GetLastResponse)           |
| GetHeader    | [GetHeaderRequest](#GetHeaderRequest)       | [GetHeaderResponse](#GetHeaderResponse)       |
| GetCommit    | [GetCommitRequest](#GetCommitRequest)       | [GetCommitResponse](#GetCommitResponse)       |
| GetEvents    | [GetEventsRequest](#GetEventsRequest)       | [GetEventsResponse](#GetEventsResponse)       |
| GetRegisters | [GetRegistersRequest](#GetRegistersRequest) | [GetRegistersResponse](#GetRegistersResponse) |

## Types

### GetFirstRequest

For now, `GetFirstRequest` is empty.

### GetFirstResponse

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

### GetLastRequest

For now, `GetLastRequest` is empty.

### GetLastResponse

| Field  | Type     | Label |
|--------|----------|-------|
| height | `uint64` |       |

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