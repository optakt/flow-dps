# Execute Cadence Script

## Description

This utility binary can be used to execute a Cadence script against a running Flow network. By default, access nodes are expected on `127.0.0.1:3569`. Scripts can be executed on any arbitrary block height.

## Usage

```sh
Usage of ./execute-cadence-script:
  -a, --api string           access node API address (default: "127.0.0.1:3569")
  -h, --height int           height on which to execute script, -1 for last indexed height (default: -1)
  -l, --log string           log level for JSON logger (default: "info")
  -s, --script string        cadence script to execute (required)
```

## Example

The following example executes a Cadence script to retrieve the balance of the account '0x631e88ae7f1d7c20' at two different block heights.

```console
$ ./execute-cadence-script -s get_balance.ca
100.00100004

$ ./execute-cadence-script -h 90 -s get_balance.ca
100.00100005

$ cat ./get_balance.ca
// This script reads the balance field of an account's FlowToken Balance

import FungibleToken from 0x9a0766d93b6608b7
import FlowToken from 0x7e60df042a9c0868

pub fun main(): UFix64 {

  // note that the account address is currently 
  // in the script itself, provided to the util
  var account: Address = 0x631e88ae7f1d7c20

  let vaultRef = getAccount(account)
    .getCapability(/public/flowTokenBalance)
    .borrow<&FlowToken.Vault{FungibleToken.Balance}>()
    ?? panic("Could not borrow Balance reference to the Vault")

  return vaultRef.balance
}
```

## TODOs:

- internally, height argument is `int64`, not `uint64`. this was done so that `-1` height can be used as the sentinel value for `latest` height
- templating for variable substitution?