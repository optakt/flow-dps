# Flow DPS Executor

## Description

The Flow DPS Executor provides access to Flow DPS Server's index through a REST API.
It can be used to execute Cadence scripts at an arbitrary block height of a fork.
It uses the Flow DPS Server's GRPC API as the backend to query the required data.

## Running the Executable

```sh
Usage of flow-dps-executor:
  -a, --api string     host for GRPC API server
  -l, --level string   log output level (default "info")
  -p, --port uint16    port to host Executor API on (default 8080)
```

## Usage over HTTP

The Flow DPS Executor accepts POST requests at the `/execute` endpoint.
Example of a correct HTTP request payload sent to the Executor is shown below:

```json
{
    "height": 10,
    "script": "script text",
    "arguments": [
        "type1(value1)",
        "type2(value2)",
        "type3(value3)" 
    ]
}
```

If the provided script expects no arguments, the `arguments` string array can be ommitted.
Example of an HTTP response payload sent by the Executor after successful script execution is shown below:

```json
{
    "height": 10,
    "script": "script text",
    "arguments": [
        "type1(value1)",
        "type2(value2)",
        "type3(value3)" 
    ],
    "result": <script result>
}
```

## Examples

### Running a Script With No Arguments

The example below describes an interaction where a simple script returning a single value is executed.

Script text:

```
// This script returns a hardcoded value.
// It can be used to showcase execution of scripts without arguments.

pub fun main(): UFix64 {

    let x: UFix64 = 17.0
    return x
}
```

Example of a POST request payload to execute the shown script:

```json
{
    "height": 14,
    "script": "pub fun main(): UFix64 {\n        let x: UFix64 = 17.0\n        return x\n    }"
}
```

Example of the response:

```json
{
    "height": 14,
    "script": "pub fun main(): UFix64 {\n        let x: UFix64 = 17.0\n        return x\n    }",
    "result": 1700000000
}
```
### Runinning a Script With Arguments

Script example:

```
// This script accepts a single numeric argument, and returns that argument increased by a 1000.
// This script can be used to showcase execution of a script with arguments. 

pub fun main(value: UFix64): UFix64 {

    let x: UFix64 = 1000.0 + value
    return x
}
```

Example of a POST request payload to execute the shown script:

```json
{
    "height": 45,
    "script": "pub fun main(value: UFix64): UFix64 {\n        let x: UFix64 = 1000.0 + value\n        return x\n    }",
    "arguments": [
        "UFix64(17.0)"
    ]
}
```

Example of the response:

```json
{
    "height": 45,
    "script": "pub fun main(value: UFix64): UFix64 {\n        let x: UFix64 = 1000.0 + value\n        return x\n    }",
    "arguments": [
        "UFix64(17.0)"
    ],
    "result": 101700000000
}
```
### Real World Example - Return the Balance of an Account

Script text:

```
// This script reads the balance field of an account's FlowToken Balance

import FungibleToken from 0x9a0766d93b6608b7
import FlowToken from 0x7e60df042a9c0868

pub fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .getCapability(/public/flowTokenBalance)
        .borrow<&FlowToken.Vault{FungibleToken.Balance}>()
        ?? panic("Could not borrow Balance reference to the Vault")

    return vaultRef.balance
}
```

Example of a POST request payload to execute the shown script:

```json
{
    "height": 425,
    "script": "<script text trimmed for brevity>",
    "arguments": [
        "Address(754aed9de6197641)"
    ]
}
```

Example of the response:

```json
{
    "height": 425,
    "script": "<script text trimmed for brevity>",
    "arguments": [
        "Address(754aed9de6197641)"
        ],
    "result": 10000100002
}
```

Changing the `height` argument would retrieve the balance at a different point in chain history.
