package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type AddLiquidityData struct {
	Coin0          types.CoinID
	Coin1          types.CoinID
	Volume0        *big.Int
	MaximumVolume1 *big.Int
}

func (data AddLiquidityData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin1 == data.Coin0 {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.Coin0.String(),
				data.Coin1.String(), "", "")),
		}
	}

	coin0 := context.Coins().GetCoin(data.Coin0)
	if coin0 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin0.String())),
		}
	}

	coin1 := context.Coins().GetCoin(data.Coin1)
	if coin1 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin1.String())),
		}
	}

	return nil
}

func (data AddLiquidityData) String() string {
	return fmt.Sprintf("ADD SWAP POOL")
}

func (data AddLiquidityData) Gas() int64 {
	return commissions.AddSwapPoolData
}

func (data AddLiquidityData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, priceCoin types.CoinID, price *big.Int) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	neededAmount1 := new(big.Int).Set(data.MaximumVolume1)

	swapper := checkState.Swap().GetSwapper(data.Coin0, data.Coin1)
	if swapper.IsExist() {
		if isGasCommissionFromPoolSwap {
			if tx.GasCoin == data.Coin0 && data.Coin1.IsBaseCoin() {
				swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
			}
			if tx.GasCoin == data.Coin1 && data.Coin0.IsBaseCoin() {
				swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
			}
		}
		_, neededAmount1 = swapper.CalculateAddLiquidity(data.Volume0)
		if neededAmount1.Cmp(data.MaximumVolume1) == 1 {
			return Response{
				Code: code.InsufficientInputAmount,
				Log:  fmt.Sprintf("You wanted to add %s %s, but currently you need to add %s %s to complete tx", data.Volume0, checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), neededAmount1, checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientInputAmount(data.Coin0.String(), data.Volume0.String(), data.Coin1.String(), data.MaximumVolume1.String(), neededAmount1.String())),
			}
		}
	}

	if err := swapper.CheckMint(data.Volume0, neededAmount1); err != nil {
		if err == swap.ErrorInsufficientLiquidityMinted {
			if !swapper.IsExist() {
				return Response{
					Code: code.InsufficientLiquidityMinted,
					Log: fmt.Sprintf("You wanted to add less than minimum liquidity, you should add %s %s and %s or more %s",
						"10", checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), "10", checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientLiquidityMinted(data.Coin0.String(), "10", data.Coin1.String(), "10")),
				}
			} else {
				amount0, amount1 := swapper.Amounts(big.NewInt(1))
				return Response{
					Code: code.InsufficientLiquidityMinted,
					Log: fmt.Sprintf("You wanted to add less than one liquidity, you should add %s %s and %s %s or more",
						amount0, checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), amount1, checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientLiquidityMinted(data.Coin0.String(), amount0.String(), data.Coin1.String(), amount1.String())),
				}
			}
		} else if err == swap.ErrorInsufficientInputAmount {
			return Response{
				Code: code.InsufficientInputAmount,
				Log:  fmt.Sprintf("You wanted to add %s %s, but currently you need to add %s %s to complete tx", data.Volume0, checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), neededAmount1, checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientInputAmount(data.Coin0.String(), data.Volume0.String(), data.Coin1.String(), data.MaximumVolume1.String(), neededAmount1.String())),
			}
		}
	}

	{
		amount0 := new(big.Int).Set(data.Volume0)
		if tx.GasCoin == data.Coin0 {
			amount0.Add(amount0, commission)
		}
		if checkState.Accounts().GetBalance(sender, data.Coin0).Cmp(amount0) == -1 {
			symbol := checkState.Coins().GetCoin(data.Coin0).GetFullSymbol()
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), symbol),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), amount0.String(), symbol, data.Coin0.String())),
			}
		}
	}

	{
		maximumVolume1 := new(big.Int).Set(neededAmount1)
		if tx.GasCoin == data.Coin1 {
			maximumVolume1.Add(maximumVolume1, commission)
		}
		if checkState.Accounts().GetBalance(sender, data.Coin1).Cmp(maximumVolume1) == -1 {
			symbol := checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), maximumVolume1.String(), symbol),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), maximumVolume1.String(), symbol, data.Coin1.String())),
			}
		}
	}
	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	takenAmount1 := neededAmount1
	takenLiquidity := new(big.Int).Sqrt(new(big.Int).Mul(takenAmount1, data.Volume0))
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		amount0, amount1, liquidity := deliverState.Swap.PairMint(sender, data.Coin0, data.Coin1, data.Volume0, data.MaximumVolume1)
		deliverState.Accounts.SubBalance(sender, data.Coin0, amount0)
		deliverState.Accounts.SubBalance(sender, data.Coin1, amount1)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		takenAmount1 = amount1
		takenLiquidity = liquidity
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeAddLiquidity)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.volume1"), Value: []byte(takenAmount1.String())},
		kv.Pair{Key: []byte("tx.liquidity"), Value: []byte(takenLiquidity.String())},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
