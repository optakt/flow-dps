package scripts

const (
	PlaceholderFungible = "FUNGIBLE_ADDRESS"
	PlaceholderToken    = "TOKEN_ADDRESS"
)

const GetBalance = `
// This script reads the balance field of an account's FlowToken Balance

import FungibleToken from 0xFUNGIBLE_ADDRESS
import FlowToken from 0xTOKEN_ADDRESS

pub fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .getCapability(/public/flowTokenBalance)
        .borrow<&FlowToken.Vault{FungibleToken.Balance}>()
        ?? panic("Could not borrow Balance reference to the Vault")

    return vaultRef.balance
}
`
