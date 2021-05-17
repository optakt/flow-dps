# API Documentation

**Table of Contents**

1. [REST API](#rest-api)
   1. [`GET /registers/:raw_key` - Get Register](#get-registersraw_key---get-register)
      1. [Path Parameters](#path-parameters)
      2. [Query Parameters](#query-parameters)
      3. [Response Codes](#response-codes)
      4. [Response Body](#response-body)
   2. [`GET /values/:ledger_key` - Get Value](#get-valuesledger_key---get-value)
      1. [Path Parameters](#path-parameters-1)
      2. [Query Parameters](#query-parameters-1)
      3. [Response Codes](#response-codes-1)
      4. [Response Body](#response-body-1)
2. [Rosetta API](#rosetta-api)
3. [GRPC API](#grpc-api)

## REST API

### `GET /registers/:raw_key` - Get Register

This route returns the raw binary payload, encoded in hexadecimal, for the key's register in the execution state trie.

**Example request**: `GET /registers/6c49490a1f023fda632cfe3a49b662016c49490a1f023fda632cfe3a49b66201?height=425`

#### Path Parameters

* `raw_key`: The hexadecimal-encoded key at which to look for a register.

#### Query Parameters

* `height`: Optional. The height at which to look for the key's payload. Defaults to the height of the last sealed block indexed by the service.

#### Response Codes

Possible response codes are:

* `200 OK` - Payload retrieved successfully.
* `400 Bad Request` - Unable to decode key or height parameters.
* `404 Not Found` - Register key not found at the specified height, or at the last sealed height, if no height given.
* `500 Internal Server Error` - Unable to create query or to read from state database.

#### Response Body

**JSON Schema:**

```json
{
  "title": "Register value response",
  "type": "object",
  "properties": {
    "height": {
      "type": "uint64",
      "description": "The height at which a payload was found."
    },
    "key": {
      "type": "string",
      "description": "The hex-encoded key of the register."
    },
    "value": {
      "type": "string",
      "description": "The payload of the register."
    }
  }
}
```

**Example response:**

```json
{
  "height": 425,
  "key": "3982123eefe952df5900dbabac2771e4",
  "value": "42c42176e24d2eeb4f9dfbee38c727f75ed07d8e4623863d350a523ceab36411829560e1848aa47aef4bf3bca6cb86c437561fc5d6afb7fa62b0a17559e4f7eb4be8"
}
```

### `GET /values/:ledger_key` - Get Value

This route returns the Ledger payload for a Ledger key from the execution state trie.
The encoding of the key is inspired by the canonical string format for keys, but made compatible with URLs and with requests for multiple keys.

**Example request**: `GET /values/0.f647acg,4.ef67d11:0.f3321ab,3.ab321fe?hash=7ae6417ed5&version=1`

#### Path Parameters

* `ledger_key`: A semicolon-delimited (`:`) set of `ledger.Key` strings. Each `ledger.KeyPart` within the `ledger.Key` is delimited by a comma (`,`). The type and value of each `ledger.KeyPart` are delimited by a dot (`.`), and the values are encoded as hexadecimal strings.

#### Query Parameters

* `hash`: Optional. Specifies which state commitment hash to get the payload value from. Defaults to the state commitment of the last sealed block indexed by the service.
* `version`: Optional. Specifies the pathfinder version to use to traverse the state trie. Defaults to the default pathfinder of the Flow Go dependency version.

#### Response Codes

Possible response codes are:

* `200 OK` - Payload retrieved successfully.
* `400 Bad Request` - Unable to decode key, hash or version parameters.
* `404 Not Found` - No payload found for given parameters.
* `500 Internal Server Error` - Unable to create query or to read from state database.

#### Response Body

**JSON Schema:**

```json
{
  "title": "Payload Value of an encoded Ledger entry",
  "type": "array",
  "items": {
    "type": "string"
  }
}
```

**Example response:**

```json
[
  "6c49490a1f023fda632cfe3",
  "24e32f4633ff12daf66f1e2d8c73b04f",
  "7bc1e622a5b639e8befe97262d3a",
  "1d7dd90eca1066a5905abf243b926d35",
  "8c5178bcaa7b30cec5c"
]
```

## Rosetta API

The Rosetta API follows the Rosetta Data API specification. Refer to the [official documentation](https://www.rosetta-api.org/docs/data_api_introduction.html) API schemas.

The implemented API endpoints are:

* [Block API](https://www.rosetta-api.org/docs/BlockApi.html)
* [Account API](https://www.rosetta-api.org/docs/AccountApi.html)

## GRPC API

TODO
