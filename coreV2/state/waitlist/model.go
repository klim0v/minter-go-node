package waitlist

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"sync"
)

type Item struct {
	CandidateId uint32
	Coin        types.CoinID
	Value       *big.Int
}

type Model struct {
	List []*Item

	address   types.Address
	markDirty func(address types.Address)
	lock      sync.RWMutex
}

func (m *Model) Address() types.Address {
	return m.address
}

func (m *Model) AddToList(candidateId uint32, coin types.CoinID, value *big.Int) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, item := range m.List {
		if item.Coin == coin && item.CandidateId == candidateId {
			item.Value.Add(item.Value, value)
			return
		}
	}
	m.List = append(m.List, &Item{
		CandidateId: candidateId,
		Coin:        coin,
		Value:       new(big.Int).Set(value),
	})
}
