# API Documentation

## Table of Contents

1. [Rest API](#rest-api)
    1.1 [GET /registers/:key](#get-registerskey---get-register)
    1.2 [GET /values/:encoded_key](#get-valuesencoded_key---get-value)
2. [Rosetta API](#rosetta-api)
3. [GRPC API](#grpc-api)

## REST API

### `GET /registers/:key` - Get Register

This route returns a register's payload from its key.

**Example request**: `GET /registers/6c49490a1f023fda632cfe3a49b662016c49490a1f023fda632cfe3a49b66201?height=425`

#### Path Parameters

* `key`: The hexadecimal-encoded key at which to look for a register.

#### Query Parameters

* `height`: Optional. The height at which to look for the key's payload. If no value is found that the given height, the last height at which a value was set for this register is used. Defaults to `0`.

#### Response Codes

Possible response codes are:

* `200 OK` - Payload retrieved successfully.
* `400 Bad Request` - Unable to decode key or height parameters.
* `404 Not Found` - No payload found for given parameters.
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
  "key": "20VDBpq8GZS3JDdmskD70Deq+kdsNetzeCZA=",
  "value": "+Em4QOD6BU2j9jPdks07DsmalPEDfbsdAk/Cdvs0Dkawo/ei02ocds9D9klSflsa8CH8p9A4ID6A6A"
}
```

### `GET /values/:encoded_key` - Get Value

This route returns the payload value of an encoded Ledger entry.

**Example request**: `GET /values/0.f647acg,4.ef67d11:0.f3321ab,3.ab321fe?hash=7ae6417ed5&version=1`

#### Path Parameters

* `encoded_key`: A semicolon-delimited (`:`) set of `ledger.Key` strings. Each `ledger.KeyPart` within the `ledger.Key` is delimited by a comma (`,`). The type and value of each `ledger.KeyPart` are delimited by a dot (`.`), and the values are encoded as hexadecimal strings.

#### Query Parameters

* `hash`: Optional. Specifies which commit hash to get the payload value from. Defaults to the latest commit from the state.
* `version`: Optional. Specifies the pathfinding version to use to traverse the state. Defaults to the default pathfinder key encoding.

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
  "6c49490a1f023fda632cfe3a49b66201",
  "24e32f4633ff12daf66f1e2d8c73b04f",
  "7bc1e622a5b639e8befe97262d3a21c5",
  "1d7dd90eca1066a5905abf243b926d35",
  "8c5178bcaa7b30cec5c9073aee1a1702"
]
```

## Rosetta API

The Rosetta API follows the Rosetta Data API specification. Refer to the [official documentation](https://www.rosetta-api.org/docs/data_api_introduction.html) API schemas.

The implemented API endpoints are:

* [Block API](https://www.rosetta-api.org/docs/BlockApi.html)
* [Account API](https://www.rosetta-api.org/docs/AccountApi.html)

## GRPC API

TODO