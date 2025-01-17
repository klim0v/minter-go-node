package tests_old

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
	"time"
)

func TestReward_Simple(t *testing.T) {
	state := DefaultAppState() // generate default state

	stake := helpers.BipToPip(big.NewInt(10_000)).String()

	state.Validators = []types.Validator{
		{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{1},
			AccumReward:   "1000000",
			AbsentTimes:   types.NewBitArray(24),
		},
	}

	state.Candidates = []types.Candidate{
		{
			ID:             1,
			RewardAddress:  types.Address{1},
			OwnerAddress:   types.Address{1},
			ControlAddress: types.Address{1},
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{1},
			Commission:     5,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{5},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates:                  nil,
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		},
	}
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 10,
			Votes: []types.Pubkey{
				[32]byte{1},
			},
			Version: "v310",
		},
	}
	state.Coins = []types.Coin{
		{
			ID:           types.USDTID,
			Name:         "USDT (Tether USD, Ethereum)",
			Symbol:       types.StrToCoinBaseSymbol("USDTE"),
			Volume:       "10000000000000000000000000",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "10000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{},
			Mintable:     true,
			Burnable:     true,
		},
	}
	state.Pools = []types.Pool{
		{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "3500000000000000000000000000",
			Reserve1: "10000000000000000000000000",
			ID:       1,
			Orders:   nil,
		},
	}

	app := CreateApp(state, 9) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 11) // send BeginBlock
	SendEndBlock(app, 11)   // send EndBlock
	SendCommit(app)         // send Commit

	t.Log(app.UpdateVersions()[1])
	//t.Log(app.GetEventsDB().LoadEvents(11)[0])
	t.Log(app.CurrentState().App().Reward())
}

func TestReward_X3Cmp(t *testing.T) {
	t.Skip("move to tests_new")
	state := DefaultAppState() // generate default state

	stake := helpers.BipToPip(big.NewInt(10_000))

	state.Accounts = []types.Account{
		{
			Address:             types.Address{11},
			Balance:             nil,
			Nonce:               0,
			MultisigData:        nil,
			LockStakeUntilBlock: 99999999,
		},
	}
	state.Validators = []types.Validator{
		{
			TotalBipStake: big.NewInt(0).Add(stake, stake).String(),
			PubKey:        types.Pubkey{1},
			AccumReward:   "1000000",
			AbsentTimes:   types.NewBitArray(24),
		},
	}

	state.Candidates = []types.Candidate{
		{
			ID:             1,
			RewardAddress:  types.Address{1},
			OwnerAddress:   types.Address{1},
			ControlAddress: types.Address{1},
			TotalBipStake:  big.NewInt(0).Add(stake, stake).String(),
			PubKey:         types.Pubkey{1},
			Commission:     99,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{5},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
				{
					Owner:    types.Address{11},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
			},
			Updates:                  nil,
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		},
	}
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 11,
			Votes: []types.Pubkey{
				[32]byte{1},
			},
			Version: "v300",
		},
	}
	state.Coins = []types.Coin{
		{
			ID:           types.USDTID,
			Name:         "USDT (Tether USD, Ethereum)",
			Symbol:       types.StrToCoinBaseSymbol("USDTE"),
			Volume:       "10000000000000000000000000",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "10000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{},
			Mintable:     true,
			Burnable:     true,
		},
	}
	state.Pools = []types.Pool{
		{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "3500000000000000000000000000",
			Reserve1: "10000000000000000000000000",
			ID:       1,
			Orders:   nil,
		},
	}

	app := CreateApp(state, 9) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 11) // send BeginBlock
	SendEndBlock(app, 11)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 12) // send BeginBlock
	SendEndBlock(app, 12)   // send EndBlock
	SendCommit(app)         // send Commit

	t.Log(app.UpdateVersions()[1])
	t.Log(app.GetEventsDB().LoadEvents(12)[0])
	t.Log(app.CurrentState().App().Reward())

	SendBeginBlock(app, 13) // send BeginBlock
	SendEndBlock(app, 13)   // send EndBlock
	SendCommit(app)         // send Commit
	SendBeginBlock(app, 14) // send BeginBlock
	SendEndBlock(app, 14)   // send EndBlock
	SendCommit(app)         // send Commit

	t.Log(app.GetEventsDB().LoadEvents(14)[0])
	t.Log(app.GetEventsDB().LoadEvents(14)[1])
	t.Log(app.GetEventsDB().LoadEvents(14)[2])
	t.Log(app.GetEventsDB().LoadEvents(14)[3])
	t.Log(app.GetEventsDB().LoadEvents(14)[4])
	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	t.Logf("%#v", appState.Candidates[0].Stakes[0].Value) // 10001152000000000000000 +
	t.Logf("%#v", appState.Candidates[0].Stakes[1].Value) // 10001152000000000000000 + x3
	if big.NewInt(0).Div(big.NewInt(0).Sub(helpers.StringToBigInt(appState.Candidates[0].Stakes[1].Value), helpers.StringToBigInt("10001152000000000000000")),
		big.NewInt(0).Sub(helpers.StringToBigInt(appState.Candidates[0].Stakes[0].Value), helpers.StringToBigInt("10001152000000000000000"))).
		String() != "3" {
		t.Error("ee")
	}
}

func TestReward_Update_Down(t *testing.T) {
	state := DefaultAppState() // generate default state

	address, pk := CreateAddress() // create account for test

	state.Accounts = []types.Account{
		{
			Address: address,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("1000000000300000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		{
			Address:             types.Address{11},
			Balance:             nil,
			Nonce:               0,
			MultisigData:        nil,
			LockStakeUntilBlock: 99999999,
		},
	}
	stake := helpers.BipToPip(big.NewInt(10_000))
	totalBipStake := big.NewInt(0).Add(stake, stake)
	state.Validators = []types.Validator{
		{
			TotalBipStake: totalBipStake.String(),
			PubKey:        types.Pubkey{1},
			AccumReward:   "1000000",
			AbsentTimes:   types.NewBitArray(24),
		},
	}

	state.Candidates = []types.Candidate{
		{
			ID:             1,
			RewardAddress:  types.Address{1},
			OwnerAddress:   types.Address{1},
			ControlAddress: types.Address{1},
			TotalBipStake:  totalBipStake.String(),
			PubKey:         types.Pubkey{1},
			Commission:     5,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{5},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
				{
					Owner:    types.Address{11},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
			},
			Updates:                  nil,
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		},
	}
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 10,
			Votes: []types.Pubkey{
				[32]byte{1},
			},
			Version: "v300",
		},
	}
	state.Coins = []types.Coin{
		{
			ID:           types.USDTID,
			Name:         "USDT (Tether USD, Ethereum)",
			Symbol:       types.StrToCoinBaseSymbol("USDTE"),
			Volume:       "10000000000000000000000000",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "10000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{},
			Mintable:     true,
			Burnable:     true,
		},
	}
	state.Pools = []types.Pool{
		{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "3500000000000000000000000000",
			Reserve1: "10000000000000000000000000",
			ID:       1,
			Orders:   nil,
		},
	}

	app := CreateApp(state, 9) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 11, time.Unix(1643889603, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 11)                                   // send EndBlock
	SendCommit(app)                                         // send Commit
	{
		SendBeginBlock(app, 12) // send BeginBlock

		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV260{
			Coins:             []types.CoinID{0, types.USDTID},
			ValueToSell:       helpers.StringToBigInt("1000000000000000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

		SendEndBlock(app, 12) // send EndBlock
		SendCommit(app)       // send Commit
	}

	SendBeginBlock(app, 13, time.Unix(1646308803, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 13)                                   // send EndBlock
	SendCommit(app)
	SendBeginBlock(app, 14, time.Unix(1646308803, 0).UTC()) // send BeginBlock
	{
		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV260{
			Coins:             []types.CoinID{0, types.USDTID},
			ValueToSell:       helpers.StringToBigInt("100000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)
		response := SendTx(app, SignTx(pk, tx)) // compose and send tx
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	SendEndBlock(app, 14) // send EndBlock
	SendCommit(app)       // send Commit

	t.Log(app.GetEventsDB().LoadEvents(13)[0])

	t.Log(app.GetEventsDB().LoadEvents(14)[0])
	t.Log(app.GetEventsDB().LoadEvents(14)[1])
	t.Log(app.GetEventsDB().LoadEvents(14)[2])
	t.Log(app.GetEventsDB().LoadEvents(14)[3])
	//t.Log(app.GetEventsDB().LoadEvents(14)[4])
	t.Log(app.CurrentState().App().Reward())

	SendBeginBlock(app, 15, time.Unix(1650628838, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 15)                                   // send EndBlock
	SendCommit(app)

	t.Log(app.CurrentState().App().Reward())
}

func TestReward_Update_Up(t *testing.T) {
	state := DefaultAppState() // generate default state

	address, pk := CreateAddress() // create account for test

	state.Accounts = []types.Account{
		{
			Address: address,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("1000000000100000000000000000").String(),
				},
				{
					Coin:  uint64(types.USDTID),
					Value: helpers.StringToBigInt("10000000000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		{
			Address:             types.Address{11},
			Balance:             nil,
			Nonce:               0,
			MultisigData:        nil,
			LockStakeUntilBlock: 99999999,
		},
	}
	stake := helpers.BipToPip(big.NewInt(10_000))
	totalBipStake := big.NewInt(0).Add(stake, stake)
	state.Validators = []types.Validator{
		{
			TotalBipStake: totalBipStake.String(),
			PubKey:        types.Pubkey{1},
			AccumReward:   "1000000",
			AbsentTimes:   types.NewBitArray(24),
		},
	}

	state.Candidates = []types.Candidate{
		{
			ID:             1,
			RewardAddress:  types.Address{1},
			OwnerAddress:   types.Address{1},
			ControlAddress: types.Address{1},
			TotalBipStake:  totalBipStake.String(),
			PubKey:         types.Pubkey{1},
			Commission:     5,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{5},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
				{
					Owner:    types.Address{11},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
			},
			Updates:                  nil,
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		},
	}
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 10,
			Votes: []types.Pubkey{
				[32]byte{1},
			},
			Version: "v300",
		},
	}
	state.Coins = []types.Coin{
		{
			ID:           types.USDTID,
			Name:         "USDT (Tether USD, Ethereum)",
			Symbol:       types.StrToCoinBaseSymbol("USDTE"),
			Volume:       "20000000000000000000000000",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "10000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{},
			Mintable:     true,
			Burnable:     true,
		},
	}
	state.Pools = []types.Pool{
		{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "3500000000000000000000000000",
			Reserve1: "10000000000000000000000000",
			ID:       1,
			Orders:   nil,
		},
	}

	app := CreateApp(state, 9) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 11, time.Unix(1643716803, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 11)                                   // send EndBlock
	SendCommit(app)                                         // send Commit
	{
		SendBeginBlock(app, 12) // send BeginBlock

		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
			Coins:             []types.CoinID{types.USDTID, 0},
			ValueToSell:       helpers.StringToBigInt("1000000000000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

		SendEndBlock(app, 12) // send EndBlock
		SendCommit(app)       // send Commit
	}

	SendBeginBlock(app, 13, time.Unix(1643803203, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 13)                                   // send EndBlock
	SendCommit(app)                                         // send Commit

	SendBeginBlock(app, 14, time.Unix(1643803203, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 14)                                   // send EndBlock
	SendCommit(app)                                         // send Commit

	t.Log(app.GetEventsDB().LoadEvents(11)[0])
	t.Log(app.GetEventsDB().LoadEvents(14)[0])
	t.Log(app.GetEventsDB().LoadEvents(14)[1])
	t.Log(app.GetEventsDB().LoadEvents(14)[2])
	t.Log(app.GetEventsDB().LoadEvents(14)[3])
	t.Log(app.GetEventsDB().LoadEvents(14)[4])
	t.Log(app.GetEventsDB().LoadEvents(14)[5])
	t.Log(app.GetEventsDB().LoadEvents(14)[6])
	t.Log(app.GetEventsDB().LoadEvents(14)[7])

	{
		SendBeginBlock(app, 14) // send BeginBlock

		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
			Coins:             []types.CoinID{types.USDTID, 0},
			ValueToSell:       helpers.StringToBigInt("5000000000000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

		SendEndBlock(app, 14) // send EndBlock
		SendCommit(app)       // send Commit
	}

	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}

	t.Logf("%#v", appState.Candidates[0].Stakes[0])
	t.Logf("%#v", appState.Candidates[0].Stakes[1])

	SendBeginBlock(app, 15, time.Unix(1643889603, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 15)                                   // send EndBlock
	SendCommit(app)                                         // send Commit

	SendBeginBlock(app, 16, time.Unix(1643889603, 0).UTC()) // send BeginBlock
	SendEndBlock(app, 16)                                   // send EndBlock
	SendCommit(app)                                         // send Commit

	t.Log(app.GetEventsDB().LoadEvents(15)[0])

	appState = app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	t.Logf("%#v", appState.Candidates[0].Stakes[0])
	t.Logf("%#v", appState.Candidates[0].Stakes[1])

}
