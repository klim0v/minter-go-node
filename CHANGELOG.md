# Changelog

## [v3.4.0](https://github.com/MinterTeam/minter-go-node/tree/v3.4.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v3.3.0...v3.4.0)

### Added

- API methods: `waitlist_all`, `limit_orders_by_owner`
- Indexed tags to search for transactions by order
- Starting a node without processing new states with tag `--only-api-mode`

### Fixed

- API Balance deadlock
- Calculation of x3 rewards

## [v3.3.0](https://github.com/MinterTeam/minter-go-node/tree/v3.3.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v3.2.0...v3.3.0)

### Fixed

- Accruals for DAOs and developers, taking into account blocked stakes

## [v3.2.0](https://github.com/MinterTeam/minter-go-node/tree/v3.2.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v3.1.1...v3.2.0)

### Fixed

- Smooth increase in rewards after the fall
- Accruals for DAOs and developers, taking into account blocked stakes

## [v3.1.1](https://github.com/MinterTeam/minter-go-node/tree/v3.1.1)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v3.1.0...v3.1.1)

### Fixed

- Find coins with last symbol `-`
- Accrual of rewards x3 with candidate's `AccumReward` is 0

## [v3.1.0](https://github.com/MinterTeam/minter-go-node/tree/v3.1.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v3.0.3...v3.1.0)

### Fixed

- Accrual of blocked rewards
- `state-sync` default config params

## [v3.0.3](https://github.com/MinterTeam/minter-go-node/tree/v3.0.2)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v3.0.0...v3.0.2)

### Fixed

- Initial emission
- API `/block` and `/blocks` for older versions.

## [v3.0.0](https://github.com/MinterTeam/minter-go-node/tree/v3.0.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.6.1...v3.0.0)

### Added

- Dynamic Mining
- Auto-Redelegation
- Burning by Balancer
- Burning of Ticker Fees
- BIP Buyback
- 3-Year BIP Lock
- Stake Transfer

### Fixed

- Limit order caching error

## [v2.6.1](https://github.com/MinterTeam/minter-go-node/tree/v2.6.1)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.6.0...v2.6.1)

### Added

- Support for state sync snapshots

### Fixed

- Bug evaluating the exchange of coins through a bankor in API `/v2/estimate_coin_sell_all`

## [v2.6.0](https://github.com/MinterTeam/minter-go-node/tree/v2.6.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.5.0...v2.6.0)

### Added

- AMM with Order Book
- Allow all holders to burn tokens
- Limitation of the maximum total stake of a candidate to 20% of the sum of all candidates
- Removal of candidates that do not fall into the top 100 by stake size

## [v2.5.0](https://github.com/MinterTeam/minter-go-node/tree/v2.5.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.4.1...v2.5.0)

### Added

- Commission fee for unsuccessful transactions
- Prioritizing transactions in a block based on gas multiplier (not enabled)

### Fixed

- Custom commissions with gas multiplier
- Parallel getting and updating of balances

## [v2.4.1](https://github.com/MinterTeam/minter-go-node/tree/v2.4.1)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.4.0...v2.4.1)

### Fixed

- Deadlock in block's commit
- Backward compatibility of older blocks

## [v2.4.0](https://github.com/MinterTeam/minter-go-node/tree/v2.4.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.3.0...v2.4.0)

### Fixed

- Checks for transactions

## [v2.3.0](https://github.com/MinterTeam/minter-go-node/tree/v2.3.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.2.1...v2.3.0)

### Fixed

- Bug with payment of commission by LP-tokens

## [v2.2.1](https://github.com/MinterTeam/minter-go-node/tree/v2.2.1)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.2.0...v2.2.1)

### Fixed

- Swap pool tx tags *coin_to_sell* and *coin_to_buy*
- Hash of the transaction in send API response

## [v2.2.0](https://github.com/MinterTeam/minter-go-node/tree/v2.2.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.1.0...v2.2.0)

### Fixed

- Critical bug in the exchange transaction through the pool
- Import and export for new entities
- Graceful stop of the node

## [v2.1.0](https://github.com/MinterTeam/minter-go-node/tree/v2.1.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.0.3...v2.1.0)

### Fixed

- Correction of votes

## [v2.0.3](https://github.com/MinterTeam/minter-go-node/tree/v2.0.3)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.0.2...v2.0.3)

### Fixed

- Issue with db corruption while using delegated balance of addresses API
- Improved processing of incoming transactions

## [v2.0.2](https://github.com/MinterTeam/minter-go-node/tree/v2.0.2)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.0.1...v2.0.2)

### Fixed

- Fix restart bug

## [v2.0.1](https://github.com/MinterTeam/minter-go-node/tree/v2.0.1)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v2.0...v2.0.1)

### Fixed

- Issue with db corruption while using list of candidates API
- Connecting to peers with cli

### Added

- Default link to current genesis

## [v2.0.0](https://github.com/MinterTeam/minter-go-node/tree/v2.0)

[Full Changelog](https://github.com/MinterTeam/minter-go-node/compare/v1.2.1...v2.0)

### Added

- Liquidity pools
- Voting for commissions (linked to the price of an arbitrary coin)
- Smooth network update algo without stopping
- Tokens (coins without reserve)
- Change of the validator's commission
- The period for recalculating stakes and paying out rewards has been extended from 120 blocks to 720
- The penalty for the validator for missing the blocks was replaced by a 24-hour ban
- Dust pruning logic: on network update prune accounts with < 10000000 pip (0.00000000001 BIP) in balance and with no outgoing txs.
- Candidates pruning logic: automatically unbond and remove candidates which are not in the top 100
- Require coins to have at least one letter in symbol
- Fix coin ownership: set owners to a coins where >90% of supply stored in one account

### Changed

- API v2 [changelog](https://klim0v.github.io/minter-node-api-v2-diff/)
- continuation of block numbering from the previous version of the network

### Removed

- API v1  
- `PriceVote` transaction

### Security

- Update Tendermint to 0.34.8
- Update IAVL to v0.15.3

## 1.2.1

- [core] Add tags old_coin_symbol and old_coin_id to RecreateCoin tx
- [core] Fix storage leak
- [core] Update iavl 14.2
- [core] Increase CommissionMultiplier to 10e16
- [cli] Add error output instead of panic
- [api] Add the flag not_show_states for /v2/candidates
- [api] Add a /v2/test_block for testnet mode

## 1.2.0

- [core] Added ControlAddress for Candidate
- [core] Added changing candidate’s public key functionality
- [core] Coins now identified by ID, not by symbols
- [core] Added RecreateCoin tx
- [core] Added ChangeCoinOwner tx
- [core] Limit validators slots to 64
- [core] Add EditMultisigData tx
- [core] Add PriceVoteData tx
- [core] Stake value calculation changes
- [console] Added PruneBlocks command
- [api] Marked as deprecated version of API v1
- [api] Added Swagger UI for API v2

## 1.1.8

BUG FIXES

- [core] Handle coins with 0-total-valued stakes

## 1.1.7

IMPROVEMENT

- [tendermint] Upgrade to [v0.33.3](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0333)

BUG FIXES

- [dashboard] Some minor fixes

## 1.1.6

IMPROVEMENT

- [export] Added export command
- [db] Update IAVL to v0.13.2
- [console] Added dashboard command
- [docker] Fix docker build config (@dmitry-ee)

BUG FIXES

- [prometheus] Fix too many open descriptors problem

## 1.1.5

IMPROVEMENT

- [core] Check open files limits before starting the node
- [tendermint] Rollback to v0 blockchain reactor

## 1.1.4

IMPROVEMENT

- [core] Load genesis from github if not exists
- [core] Reset state on upgrades
- [config] Add `state_mem_available` param
- [api] Optimize `/candidate` endpoint
- [prometheus] Add latest block timestamp

BUG FIXES

- [config] Fix default config

## 1.1.3

BREAKING CHANGES

- [core] Fixed stakes recalculation

IMPROVEMENT

- [tendermint] Upgrade to [v0.33.2](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0332)

## 1.1.0

BREAKING CHANGES

- [core] Fix invalid check's sig issue [#264](https://github.com/MinterTeam/minter-go-node/issues/264)
- [core] Refactoring
- [core] Add Coin's MaxSupply
- [core] Remove CreatedAtBlock field in candidates
- [core] Add GasCoin to Checks
- [core] Fix buy coin commission calculation
- [core] Fix sell coin commission calculation
- [core] Enable multisignatures
- [core] Do not delete coins with small reserve
- [core] Do now allow to sell coins with reserve less than 10,000 bip
- [core] Set min coin reserve to 10,000 bip
- [core] Pay rewards each 120 blocks
- [core] Fix create coin commission issue
- [gui] Remove GUI
- [config] KeepStateHistory -> KeepLastStates
- [config] Add state_cache_size option
- [tendermint] Upgrade to [v0.33.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0331)

## 1.0.5

BUG FIXES

- [core] Fix coin liquidation issue

IMPROVEMENT

- [core] Add grace period from 4262457 to 4262500 block
- [cmd] Set start time at 7:00 AM Wednesday, January 22, 2020
- [config] Add halt_height param

## 1.0.4

IMPROVEMENT

- [tendermint] Update to [v0.32.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0321)
- [api] Add page and perPage params to /api/transactions (@Danmer)
- [cmd] Add `minter version` command

## 1.0.3

BUG FIXES

- [core] Fix coin liquidation issue

## 1.0.2

IMPROVEMENT

- [config] Add new seed nodes
- [api] Add /missed_blocks endpoint

BUG FIXES

- [api] Fix block proposer issue
- [api] Fix 0x api error
- [core] Fix "Stake is too low" issue with new custom coins
- [core] Fix redeem check invariants

## 1.0.0

BUG FIXES

- [core] Fix conversion issue

## 0.20.5

IMPROVEMENT

- [api] Speed up state history

## 0.20.4

BUG FIXES

- [core] Fix empty frozen funds case

## 0.20.3

BUG FIXES

- [api] Fix block signatures

## 0.20.2

BUG FIXES

- [api] Fix block signatures for first block

## 0.20.1

BUG FIXES

- [api] Fix current blocks signatures

## 0.20.0

IMPROVEMENT

- [core] Add remainder to total slashed
- [cmd] Add `--network-id` flag

## 0.19.2

BUG FIXES

- [core] Fix slice issues

## 0.19.0

BREAKING CHANGES

- [core] Add ChainID to transactions
- [core] Add ChainID to check
- [core] Simplify coin creation commissions (remove base commission)

IMPROVEMENT

- [api] Fix list of endpoints
- [cmd] Minter node now launched as `minter node` command

BUG FIXES

- [core] Fix incorrect coin conversion
- [tendermint] Update to [v0.31.5](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0315)

## 0.18.1

BUG FIXES

- [core] Fix attempt to double coin deletion
- [core] Add remainder to total slashed value

## 0.18.0

IMPROVEMENT

- [api] Add `/estimate_coin_sell_all` endpoint

BUG FIXES

- [p2p] Make new addressbook file for each testnet

## 0.17.1

BUG FIXES

- [core] Fix issue with candidacy declaration

## 0.17.0

BUG FIXES

- [core] Fix bug which causes dropped in first 120 blocks validators to stay in val list forever
- [core] Set start height for validators count
- [core] Add value to existing basecoin stake if exists when deleting coin instead of creating new one
- [core] Fix issue with coin deletion
- [tendermint] Update to [v0.31.3](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0313)

## 0.16.0

BREAKING CHANGES

- [core] Set min coin reserve to 1000 bip
- [core] Coins with 7-10 letters are now requires 100 bips fee
- [core] Delete coin if reserve is less than 100 bips OR price is less than 0.0001 bip, OR volume is less than 1 coin

IMPROVEMENT

- [api] Make compact json responses
- [api] Add `/genesis` endpoint
- [check] Make check's nonce a byte array field. Max 16 bytes.
- [appState] Use `startHeight` in genesis to manage rewards
- [crypto] Update crypto library
- [core] Add option to use cleveldb

BUG FIXES

- [core] Fix issue with multiple punishments to byzantine validator
- [core] Make accum reward of dropped validator distributes again between active ones
- [tendermint] Update to [v0.31.2](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0312)

## 0.15.2

IMPROVEMENT

- [cmd] `--show_validator` flag now returns hex public key of a validator
- [tendermint] Update to [v0.31.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0311)

## 0.15.1

IMPROVEMENT

- [cmd] Add `--show_node_id` and `--show_validator` flags

## 0.15.0

BREAKING CHANGES

- [tendermint] Update to [v0.31.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0310)

IMPROVEMENT

- [invariants] Add invariants checker each 720 blocks
- [core] Delete coins with 0 reserves [#217](https://github.com/MinterTeam/minter-go-node/issues/217)
- [genesis] Add option to export/import state
- [api] Add ?include_stakes to /candidates endpoint [#222](https://github.com/MinterTeam/minter-go-node/issues/222)
- [api] Change `stake` to `value` in DelegateTx
- [api] Change `pubkey` to `pub_key` in all API resources and requests
- [events] Add CoinLiquidation event [#221](https://github.com/MinterTeam/minter-go-node/issues/221)
- [mempool] Recheck mempool once per minute

BUG FIXES

- [core] Fix double sign slashing issue [#215](https://github.com/MinterTeam/minter-go-node/issues/215)
- [core] Fix issue with slashing small stake [#209](https://github.com/MinterTeam/minter-go-node/issues/209)
- [core] Fix coin creation issue
- [core] Fix mempool issue [#220](https://github.com/MinterTeam/minter-go-node/issues/220)
- [api] Make block hash lowercase [#214](https://github.com/MinterTeam/minter-go-node/issues/214)

## 0.14.3

BUG FIXES

- [core] Temp fix for consensus failure

## 0.14.2

BUG FIXES

- [events] Fix slash event on double sign (full resync needed)

## 0.14.1

IMPROVEMENT

- [api] Add /addresses endpoint
- [api] Add evidence data to /block
- [tendermint] Update to [v0.30.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0301)

BUG FIXES

- [api] Fix /block endpoint

## 0.13.1

BUG FIXES

- [core] Fix sync issue

## 0.13.0

BREAKING CHANGES

- [tendermint] Update to [v0.30.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0300)

BUG FIXES

- [core] Fix max tx length

## 0.12.1

BUG FIXES

- [core] Fix "No info about LastBlocksTimeDelta is available" issue

## 0.12.0

BREAKING CHANGES

- [core] Updated commission handling
- [core] Fix multisend issue
- [core] Extend max tx size
- [api] Fixes in error responses

## 0.11.0

BREAKING CHANGES

- [core] Fix coin convert issue
- [tendermint] Update to [v0.29.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0291)

## 0.10.1

*Jan 22th, 2019*

BREAKING CHANGES

- [tendermint] Update to [v0.29.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0290)

## 0.10.0

*Jan 20th, 2019*

BREAKING CHANGES

- [core] Add EditCandidate transaction
- [core] Make validators count logic conforms to mainnet
- [tendermint] Update to [v0.28.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0281)

BUG FIXES

- [core] Various bug fixes

IMPROVEMENT

- [mempool] Add variable min gas price threshold
- [p2p] Lower FlushThrottleTimeout to 10 ms
- [rpc] RPC errors are now delivered with 500 HTTP code
- [rpc] Prettify RPC errors

## 0.9.6

*Dec 27th, 2018*

BUG FIXES

- [core] Fix issue with corrupted db

## 0.9.5

*Dec 26th, 2018*

BUG FIXES

- [core] Fix issue with corrupted db

## 0.9.4

*Dec 26th, 2018*

IMPROVEMENT

- [mempool] Disable tx rechecking

BUG FIXES

- [core] Fix issue with bag tx occupying mempool

## 0.9.3

*Dec 25th, 2018*

BUG FIXES

- [core] Fix sell all coin tx

## 0.9.2

*Dec 25th, 2018*

BUG FIXES

- [core] Increase max block bytes

## 0.9.1

*Dec 24th, 2018*

BUG FIXES

- [api] Fix create coin tx error

## 0.9.0

*Dec 24th, 2018*

IMPROVEMENT

- [events] Refactor events
- [api] [#183](https://github.com/MinterTeam/minter-go-node/issues/183) Report if node has full state history in /status
- [api] [#164](https://github.com/MinterTeam/minter-go-node/issues/164) Add /unconfirmed_txs endpoint
- [api] Add /max_gas endpoint
- [core] Do not accept 2 transactions from same address in mempool at once
- [core] Add missing tags to transactions
- [core] Dynamically adjust max gas in blocks
- [core] Update commissions
- [tendermint] Update to v0.27.4

BUG FIXES

- [core] Fix issue with `SellAll` tx
- [core] Fix issue [#182](https://github.com/MinterTeam/minter-go-node/issues/182) with candidate owner's address
- [core] Fix max coin supply
- [api] Fix tx tags

## 0.8.5

*Dec 11th, 2018*

BUG FIXES

- [api] Fix estimate coin buy empty response
- [api] Set quotes as not necessary attribute

## 0.8.4

*Dec 10th, 2018*

BUG FIXES

- [core] Fix tx processing bug

## 0.8.3

*Dec 10th, 2018*

BUG FIXES

- [events] Fix pub key formatting in API

## 0.8.2

*Dec 10th, 2018*

BUG FIXES

- [log] Add json log format

## 0.8.1

*Dec 10th, 2018*

IMPROVEMENT

- [core] Speed-up tx processing

BUG FIXES

- [config] Change default seed node

## 0.8.0

*Dec 3rd, 2018*

BREAKING CHANGES

- [api] Switch to RPC protocol
- [api] Separate events from block in API
- [core] Fix issue with incorrect coin conversion
- [core] Limit coins supply to 1,000,000,000,000,000
- [core] Set minimal reserve and min/max coin supply in CreateCoin tx
- [core] Add MinimumValueToBuy and MaximumValueToSell to convert transactions
- [tendermint] Update to [v0.27.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0270)

IMPROVEMENT

- [logs] Add `log_format` option to config
- [events] Add UnbondEvent

## 0.7.6

*Nov 27th, 2018*

IMPROVEMENT

- [tendermint] Update to [v0.26.4](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0264)

BUG FIXES

- [node] Fix issue [#168](https://github.com/MinterTeam/minter-go-node/issues/168) with unexpected database corruption

## 0.7.5

*Nov 22th, 2018*

BUG FIXES

- [api] Fix issue in which transaction appeared in `/api/transaction` before actual execution

## 0.7.4

*Nov 20th, 2018*

BUG FIXES

- [tendermint] "Send failed" is logged at debug level instead of error
- [tendermint] Set connection config properly instead of always using default
- [tendermint] Seed mode fixes:
  - Only disconnect from inbound peers
  - Use FlushStop instead of Sleep to ensure all messages are sent before disconnecting

## 0.7.3

*Nov 18th, 2018*

BUG FIXES

- [core] More fixes on issue with negative coin reserve

## 0.7.2

*Nov 18th, 2018*

BUG FIXES

- [core] Fix issue with negative coin reserve

## 0.7.1

*Nov 16th, 2018*

IMPROVEMENT

- [tendermint] Update to [v0.26.2](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0262)

## 0.7.0

*Nov 15th, 2018*

BREAKING CHANGES

- [api] `/api/sendTransaction` is now returns only `checkTx` result. Applications are now forced to manually check if
  transaction is included in blockchain.
- [tendermint] Update to [v0.26.1](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0261)
- [core] Block hash is now 32 bytes length

IMPROVEMENT

- [core] Add `MultisendTx`
- [core] Add special cases to Formulas [#140](https://github.com/MinterTeam/minter-go-node/issues/140)
- [core] Stake unbond now instant after dropping of from 1,000st
  place [#146](https://github.com/MinterTeam/minter-go-node/issues/146)
- [p2p] Default send and receive rates are now 15mB/s
- [mempool] Set max mempool size to 10,000txs
- [gui] Small GUI improvements

## 0.6.0

*Oct 30th, 2018*

BREAKING CHANGES

- [core] Set validators limit to 100 for testnet
- [core] SetCandidateOff transaction now applies immediately
- [tendermint] Update to [v0.26.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0260)

IMPROVEMENT

- [config] Add keep_state_history option
- [api] Limit API requests

## 0.5.1

*Oct 22th, 2018*

BUG FIXES

- [core] Fixed bug with unexpected node backoff

## 0.5.0

*Oct 15th, 2018*

BREAKING CHANGES

- [core] Multisig wallets
- [core] Sub coin reserve and supply on slash
- [core] Change unbond time for testnet to 720 blocks
- [core] Limit candidates count to validatorsLimit*3 at given block
- [core] Limit delegators count to 1000 per candidate/validator

IMPROVEMENT

- [tendermint] Update to [v0.25.0](https://github.com/tendermint/tendermint/blob/master/CHANGELOG.md#v0250)

## 0.4.2

*Sept 21th, 2018*

BUG FIXES

- [api] Fix concurrent API calls

## 0.4.1

*Sept 20th, 2018*

IMPROVEMENT

- [core] Speed up synchronization

BUG FIXES

- [gui] Fix validator status

## 0.4.0

*Sept 18th, 2018*

BREAKING CHANGES

- [core] Switch Ethereum Patricia Tree to IAVL
- [core] Change consensus TimeoutCommit to 4.5 sec, TimeoutPropose to 2 sec
- [core] Now validator punished if it misses 12 of 24 last blocks

IMPROVEMENT

- [config] Add validator mode
- [api] Include events by default
- [gui] Add validator status

## 0.3.8

*Sept 17th, 2018*

BUG FIXES

- [core] Proper handle of db errors

## 0.3.7

*Sept 17th, 2018*

IMPROVEMENT

- [core] Performance update

## 0.3.6

*Sept 15th, 2018*

BUG FIXES

- [core] Critical fix

## 0.3.5

*Sept 13th, 2018*

IMPROVEMENT

- [api] Add Code and Log fields in transaction api

## 0.3.4

*Sept 13th, 2018*

IMPROVEMENT

- [api] Optimize events. WARNING! If you are using events you should re-sync blockchain from scratch.
- [api] Refactor api

## 0.3.3

*Sept 8th, 2018*

IMPROVEMENT

- [api] Add block size in bytes
- [api] [#100](https://github.com/MinterTeam/minter-go-node/issues/100) Add "events" to block response. To get events
  add ?withEvents=true to request URL. WARNING! You should sync blockchain from scratch to get this feature working

## 0.3.2

*Sept 8th, 2018*

BUG FIXES

- [core] Fix null pointer exception

## 0.3.1

*Sept 8th, 2018*

BUG FIXES

- [core] Fix shutdown issue

## 0.3.0

*Sept 8th, 2018*

BREAKING CHANGES

- [core] Validators are now updated each 120 blocks
- [core] Validators are now updated then at least one of current validators exceed 12 missed blocks
- [tendermint] Update Tendermint to v0.24.0

IMPROVEMENT

- [p2p] Add seed nodes
- [sync] Speed up synchronization
- [core] Extend max payload size to 1024 bytes
- [core] Add network id checker
- [core] Add tx.sell_amount to SellAllCoin tags
- [core] Change punishment for byzantine behavior
- [api] Limit balance watchers to 10 clients
- [config] Add config file
- [config] Add GUI listen address to config
- [config] Add API listen address to config
- [docs] Update documentation
- [validators] Remove 0-valued stakes from state

BUG FIXES

- [core] Fix issue [#77](https://github.com/MinterTeam/minter-go-node/issues/77) Incorrect createCoin fee
- [core] Fix issue with insufficient coin reserve in buy coin tx
- [core] Fix unbond transaction
- [api] Fix issue [#82](https://github.com/MinterTeam/minter-go-node/issues/82)

## 0.2.4

*Aug 24th, 2018*

BUG FIXES

- [api] Fix estimateTxCommission endpoint

IMPROVEMENT

- [gui] Minor GUI updates

## 0.2.2

*Aug 23th, 2018*

In this update we well test blockchain's hardfork. There is no need to wipe old data, just be sure to update binary
until 15000 block.

BUG FIXES

- [validators] Fix api

## 0.2.1

*Aug 23th, 2018*

In this update we well test blockchain's hardfork. There is no need to wipe old data, just be sure to update binary
until 15000 block.

BUG FIXES

- [validators] Fix validators issue

## 0.2.0

*Aug 22th, 2018*

BREAKING CHANGES

- [testnet] New testnet id
- [core] New rewards
- [core] Validators list are now updated each 12 blocks
- [core] Set DAO commission to 10%
- [core] Add Developers commission of 10%
- [core] Now stake of custom coin is calculated by selling all such staked coins
- [api] Reformatted candidates and validators endpoints
- [api] tx.return tags are now encoded as strings

IMPROVEMENT

- [tendermint] Update tendermint to 0.23.0
- [api] Add block reward to api
- [api] Add bip_value field to Stake
- [api] Add /api/candidates endpoint
- [api] Add /api/estimateTxCommission endpoint
- [gui] Minor GUI update

## 0.1.9

*Aug 19th, 2018*

BUG FIXES

- [core] Critical fix

## 0.1.8

*Aug 4th, 2018*

BUG FIXES

- [core] Critical fix

## 0.1.7

*Jule 30th, 2018*

BREAKING CHANGES

- [testnet] New testnet id

IMPROVEMENT

- [validators] Added flag ``--reset-private-validator``
- [testnet] Main validator stake is set to 1 mln MNT by default

## 0.1.6

*Jule 30th, 2018*

BREAKING CHANGES

- [testnet] New testnet id

BUG FIXES

- [core] Fixed critical bug

## 0.1.5

*Jule 28th, 2018*

BUG FIXES

- [tendermint] Update tendermint to 0.22.8
- [core] Temporary critical fix

## 0.1.4

*Jule 25th, 2018*

IMPROVEMENT

- [tendermint] Update tendermint to 0.22.6

## 0.1.3

*Jule 25th, 2018*

IMPROVEMENT

- [tendermint] Update tendermint to 0.22.5

## 0.1.0

*Jule 23th, 2018*

BREAKING CHANGES

- [core] 0.1x transaction fees
- [core] Genesis is now encapsulated in code
- [core] Add new transaction type: SellAllCoin
- [core] Add GasCoin field to transaction
- [config] New config directories
- [api] Huge API update. For more info see docs

IMPROVEMENT

- [binary] Now Minter is available as single binary. There is no need to install Tendermint
- [config] 10x default send/recv rate
- [config] Recheck after empty blocks
- [core] Check transaction nonce before adding to mempool
- [performance] Huge performance enhancement due to getting rid of network overhead between tendermint and minter
- [gui] GUI introduced! You can use it by visiting http://localhost:3000/ in your local browser

BUG FIXES

- [api] Fixed raw transaction output

## 0.0.6

*Jule 16th, 2018*

BREAKING CHANGES

- [core] Change commissions
- [testnet] New testnet id
- [core] Fix transaction decoding issue
- [core] Remove transaction ConvertCoin, add SellCoin and BuyCoin. For details see the docs.
- [core] Coin name is now limited to max 64 bytes
- [api] Update estimate exchange endpoint

IMPROVEMENT

- [api] Update transaction api
- [api] Add transaction result to block api
- [mempool] Mempool cache is disabled
- [tendermint] Updated to v0.22.4
- [versioning] Adapt Semantic Versioning https://semver.org/
- [client] Add --disable-api flag to client

## 0.0.5

*Jule 4rd, 2018*

BREAKING CHANGES

- [core] Remove Reserve Coin from coin object. All coins should be reserved with base coin
- [core] Limit tx payload and service data to 128 bytes
- [core] Fix critical issue with instant convert of 2 custom coins
- [testnet] New testnet chain id (minter-test-network-9)
- [tendermint] Switched to v0.22.0

IMPROVEMENT

- [api] Fix issue with not found coins

BUG FIXES

- [api] Fix transaction endpoint

## 0.0.4

*June 24th, 2018*

BREAKING CHANGES

- [validators] Reward now is payed each 12 blocks
- [validators] Change total "validators' power" to 100 mln
- [tendermint] Switched to v0.21.0
- [testnet] New testnet chain id
- [api] Changed */api/block* response format

IMPROVEMENT

- [docs] Updated docs

BUG FIXES

- [validators] Fixed issue with incorrect pubkey length
