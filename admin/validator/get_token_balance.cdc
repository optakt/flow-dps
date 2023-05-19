// get FLOW balance of Blocto Swap FLOW<>USDT pool
import FlowSwapPair from 0xc6c77b9f5c7a378f

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

pub fun main(): {Address: AccountInfo} {
    let poolAmounts = FlowSwapPair.getPoolAmounts()

    let accountDict: {Address: AccountInfo} = {}
    let address: Address = 0xc6c77b9f5c7a378f
    let info: AccountInfo = AccountInfo()
    info.primaryAcctBalance = poolAmounts.token1Amount
    accountDict.insert(key: address, info)

    return accountDict
}