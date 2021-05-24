package retriever

const getBalance = `
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
`
