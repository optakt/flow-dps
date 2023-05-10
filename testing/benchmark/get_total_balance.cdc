import FlowStorageFees from 0x8c5303eaa26202d6
import FungibleToken from 0x9a0766d93b6608b7
import FlowToken from 0x7e60df042a9c0868
import FlowIDTableStaking from 0x9eca2b38b18b5dfe
import LockedTokens from 0x95e019a17d0e23d7
import FlowStakingCollection from 0x95e019a17d0e23d7

import MotoGPTransfer from 0xb0e5f494fd8adfff
import FlowSwapPair from 0xd9854329b7edf136

// This script gets the TOTAL number of FLOW an account owns, across unlocked, locked, and staking.

// Adds up these numbers:

// tokens in normal account
// tokens in normal account staking
// tokens in normal account delegating
// tokens in shared account
// tokens in shared account staking
// tokens in shared account delegating

pub struct AccountInfo {
    pub(set) var primaryAcctBalance: UFix64
    pub(set) var secondaryAddress: Address?
    pub(set) var secondaryAcctBalance: UFix64
    pub(set) var stakedBalance: UFix64
    pub(set) var hasVault: Bool
    pub(set) var stakes: String

    init() {
        self.primaryAcctBalance = 0.0 as UFix64
        self.secondaryAddress = nil
        self.secondaryAcctBalance = 0.0 as UFix64
        self.stakedBalance = 0.0 as UFix64
        self.hasVault = true
        self.stakes = ""
    }
}

pub fun getStakesAndDelegations(_ account: PublicAccount) : {String:UFix64} {
    
    var allNodeInfo: [FlowIDTableStaking.NodeInfo] = []
    var allDelegateInfo: [FlowIDTableStaking.DelegatorInfo] = []
    
    // ====== Get account stake - Old Way =====
    let nodeStakerCap = account.getCapability<&{FlowIDTableStaking.NodeStakerPublic}>(
        FlowIDTableStaking.NodeStakerPublicPath)
    if let nodeStakerRef = nodeStakerCap.borrow() {
        let nodeInfo = FlowIDTableStaking.NodeInfo(nodeID: nodeStakerRef.id)
        allNodeInfo.append(nodeInfo)
    }

    // ====== Get account delegation - Old Way =====
    let delegatorCap = account.getCapability<&{FlowIDTableStaking.NodeDelegatorPublic}>(
        /public/flowStakingDelegator)
    if let delegatorRef = delegatorCap.borrow() {
        let delegatorInfo = FlowIDTableStaking.DelegatorInfo(
            nodeID: delegatorRef.nodeID,
            delegatorID: delegatorRef.id
        )
        allDelegateInfo.append(delegatorInfo)
    }

    // ====== Get account stakes and delegations =====
    var doesAccountHaveStakingCollection = FlowStakingCollection.doesAccountHaveStakingCollection(address: account.address)
    if doesAccountHaveStakingCollection {
        allNodeInfo.appendAll(FlowStakingCollection.getAllNodeInfo(address: account.address))
        allDelegateInfo.appendAll(FlowStakingCollection.getAllDelegatorInfo(address: account.address))
    }

    // ====== Shared account =====
    let lockedAccountInfoCap = account
        .getCapability<&LockedTokens.TokenHolder{LockedTokens.LockedAccountInfo}>(
            LockedTokens.LockedAccountInfoPublicPath)
    if let lockedAccountInfoRef = lockedAccountInfoCap.borrow()  {
        // ====== Get shared account stake - Old Way =====
        if let nodeID = lockedAccountInfoRef.getNodeID() {
            let nodeInfo = FlowIDTableStaking.NodeInfo(nodeID: nodeID)
            allNodeInfo.append(nodeInfo)
        }

        // ====== Get shared account delegation - Old Way =====
        if let delegatorID = lockedAccountInfoRef.getDelegatorID() {
            if let nodeID = lockedAccountInfoRef.getDelegatorNodeID() {
                let delegatorInfo = FlowIDTableStaking.DelegatorInfo(nodeID: nodeID, delegatorID: delegatorID)
                allDelegateInfo.append(delegatorInfo)
            }
        }

        // ====== Get shared account stakes and delegations =====
        doesAccountHaveStakingCollection = FlowStakingCollection.doesAccountHaveStakingCollection(address: lockedAccountInfoRef.getLockedAccountAddress())
        if doesAccountHaveStakingCollection {
            allNodeInfo.appendAll(FlowStakingCollection.getAllNodeInfo(address: lockedAccountInfoRef.getLockedAccountAddress()))
            allDelegateInfo.appendAll(FlowStakingCollection.getAllDelegatorInfo(address: lockedAccountInfoRef.getLockedAccountAddress()))
        }
    }

    // ===== Aggregate all stakes and delegations in a digestible set =====
    // deduplication between the old way and the new way will happen automatically because the result is stored in a map
    let stakes : {String:UFix64} = {}
    for nodeInfo in allNodeInfo {
        let balance =  nodeInfo.tokensStaked
                        + nodeInfo.tokensCommitted
                        + nodeInfo.tokensUnstaking
                        + nodeInfo.tokensUnstaked
                        + nodeInfo.tokensRewarded

        stakes["n:".concat(nodeInfo.id)] = balance
    }

    for delegatorInfo in  allDelegateInfo {
        let balance =  delegatorInfo.tokensStaked
                        + delegatorInfo.tokensCommitted
                        + delegatorInfo.tokensUnstaking
                        + delegatorInfo.tokensUnstaked
                        + delegatorInfo.tokensRewarded
        
        stakes["n:".concat(delegatorInfo.nodeID).concat(" d:").concat(delegatorInfo.id.toString())] = balance
    }

    return stakes
}

pub fun getAddressAccountInfo(_ address: Address): AccountInfo {
        var info: AccountInfo = AccountInfo()

        let account = getAccount(address)

        if account.storageUsed == 0 as UInt64 {
            // If the account has no storage, it has no vault
            info.hasVault = false
            return info
        }

        info.hasVault = true

        if account.getLinkTarget(/public/lockedFlowTokenReceiver) != nil {
            // skip locked accounts
            return info
        }

        // Get the primary/normal account balance (unlocked tokens)
        if let vaultRef = account.getCapability(/public/flowTokenBalance)
            .borrow<&FlowToken.Vault{FungibleToken.Balance}>(){
            info.primaryAcctBalance = vaultRef.balance
        }

        // Accounts 0xf8ded0f117ba2462 and 0x3112dbb9973ff3ad have the same locked account 0x1cb9692c064024bd and the same stake id.
        // This is expected, this was made manually. 0xf8ded0f117ba2462 has all keys revoked, and is safe to skip.
        if account.address == 0xf8ded0f117ba2462 {
            return info
        }

        if account.address == 0xc6c77b9f5c7a378f {
            info.primaryAcctBalance = info.primaryAcctBalance + FlowSwapPair.getPoolAmounts().token1Amount
        }

        if account.address == 0xa49cc0ee46c54bfb {
            info.primaryAcctBalance = info.primaryAcctBalance + MotoGPTransfer.getFlowBalance()
        }

        let lockedAccountInfoCap = account
            .getCapability<&LockedTokens.TokenHolder{LockedTokens.LockedAccountInfo}>(
                LockedTokens.LockedAccountInfoPublicPath)

        // Get secondary/locked account address and balance
        if let lockedAccountInfoRef = lockedAccountInfoCap.borrow() {
            info.secondaryAddress = lockedAccountInfoRef.getLockedAccountAddress() as Address?
            // `+ FlowStorageFees.minimumStorageReservation` is due to https://github.com/onflow/flow-core-contracts/blob/6fcd492d16186e5615d2e6589bc5b7ebce41f548/contracts/LockedTokens.cdc#L308
            info.secondaryAcctBalance = lockedAccountInfoRef.getLockedAccountBalance() + FlowStorageFees.minimumStorageReservation
        }

        // Get stakes and delegations of the account and secondary/locked account
        let stakes = getStakesAndDelegations(account)

        var stakeKey = ""
        for key in stakes.keys {
            let value = stakes[key]!
            stakeKey = stakeKey.concat(key).concat(", ")
            info.stakedBalance = info.stakedBalance + value
        }
        info.stakes = stakeKey

        return info
}

pub fun main(addresses: [Address]): {Address: AccountInfo} {
    let accountDict: {Address: AccountInfo} = {}

    for address in addresses {
        var info: AccountInfo = getAddressAccountInfo(address)
        accountDict.insert(key: address, info)
    }

    return accountDict
}
