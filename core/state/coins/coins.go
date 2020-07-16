package coins

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
	"sync"
)

const (
	mainPrefix   = byte('q')
	infoPrefix   = byte('i')
	symbolPrefix = byte('s')
)

const (
	BaseVersion types.CoinVersion = 0
)

type RCoins interface {
	Export(state *types.AppState)
	Exists(id types.CoinID) bool
	ExistsBySymbol(symbol types.CoinSymbol) bool
	SubReserve(symbol types.CoinID, amount *big.Int)
	GetCoin(id types.CoinID) *Model
	GetCoinBySymbol(symbol types.CoinSymbol) *Model
	GetSymbolInfo(symbol types.CoinSymbol) *SymbolInfo
}

type Coins struct {
	list  map[types.CoinID]*Model
	dirty map[types.CoinID]struct{}

	bus  *bus.Bus
	iavl tree.MTree

	lock sync.RWMutex
}

func NewCoins(stateBus *bus.Bus, iavl tree.MTree) (*Coins, error) {
	coins := &Coins{bus: stateBus, iavl: iavl, list: map[types.CoinID]*Model{}, dirty: map[types.CoinID]struct{}{}}
	coins.bus.SetCoins(NewBus(coins))

	return coins, nil
}

func (c *Coins) Commit() error {
	coins := c.getOrderedDirtyCoins()
	for _, id := range coins {
		coin := c.getFromMap(id)
		c.lock.Lock()
		delete(c.dirty, id)
		c.lock.Unlock()

		if coin.IsCreated() {
			ids := c.getBySymbol(coin.Symbol())
			ids = append(ids, coin.ID())

			data, err := rlp.EncodeToBytes(ids)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			c.iavl.Set(getSymbolCoinsPath(coin.Symbol()), data)
			coin.isCreated = false
		}

		if coin.IsDirty() {
			data, err := rlp.EncodeToBytes(coin)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			c.iavl.Set(getCoinPath(id), data)
			coin.isDirty = false
		}

		if coin.IsInfoDirty() {
			data, err := rlp.EncodeToBytes(coin.info)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			c.iavl.Set(getCoinInfoPath(id), data)
			coin.info.isDirty = false
		}

		if coin.IsSymbolInfoDirty() {
			data, err := rlp.EncodeToBytes(coin.symbolInfo)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			c.iavl.Set(getSymbolInfoPath(coin.Symbol()), data)
			coin.symbolInfo.isDirty = false
		}
	}

	return nil
}

func (c *Coins) GetCoin(id types.CoinID) *Model {
	return c.get(id)
}

func (c *Coins) GetSymbolInfo(symbol types.CoinSymbol) *SymbolInfo {
	return c.getSymbolInfo(symbol)
}

func (c *Coins) Exists(id types.CoinID) bool {
	if id.IsBaseCoin() {
		return true
	}

	return c.get(id) != nil
}

func (c *Coins) ExistsBySymbol(symbol types.CoinSymbol) bool {
	if symbol.IsBaseCoin() {
		return true
	}

	return c.getBySymbol(symbol) != nil
}

func (c *Coins) GetCoinBySymbol(symbol types.CoinSymbol) *Model {
	coins := c.getBySymbol(symbol.GetBaseSymbol())
	if len(coins) == 0 {
		return nil
	}

	version, err := symbol.GetVersion()
	if err != nil {
		return nil
	}

	for _, coinID := range coins {
		coin := c.get(coinID)
		if coin.Version() == version {
			return coin
		}
	}

	return nil
}

func (c *Coins) SubVolume(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).SubVolume(amount)
	c.bus.Checker().AddCoinVolume(id, big.NewInt(0).Neg(amount))
}

func (c *Coins) AddVolume(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).AddVolume(amount)
	c.bus.Checker().AddCoinVolume(id, amount)
}

func (c *Coins) SubReserve(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).SubReserve(amount)
	c.bus.Checker().AddCoin(types.GetBaseCoinID(), big.NewInt(0).Neg(amount))
}

func (c *Coins) AddReserve(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).AddReserve(amount)
	c.bus.Checker().AddCoin(types.GetBaseCoinID(), amount)
}

func (c *Coins) Create(id types.CoinID, symbol types.CoinSymbol, name string,
	volume *big.Int, crr uint, reserve *big.Int, maxSupply *big.Int, owner *types.Address,
) {
	coin := &Model{
		CName:      name,
		CCrr:       crr,
		CMaxSupply: maxSupply,
		CSymbol:    symbol,
		id:         id,
		markDirty:  c.markDirty,
		isDirty:    true,
		isCreated:  true,
		info: &Info{
			Volume:  big.NewInt(0),
			Reserve: big.NewInt(0),
			isDirty: false,
		},
	}

	if owner != nil {
		coin.symbolInfo = &SymbolInfo{
			COwnerAddress: owner,
			isDirty:       true,
		}
	}

	c.setToMap(coin.id, coin)

	coin.SetReserve(reserve)
	coin.SetVolume(volume)

	c.markDirty(coin.id)

	c.bus.Checker().AddCoin(types.GetBaseCoinID(), reserve)
	c.bus.Checker().AddCoinVolume(coin.id, volume)
}

func (c *Coins) Recreate(newID types.CoinID, symbol types.CoinSymbol,
	volume *big.Int, crr uint, reserve *big.Int, maxSupply *big.Int,
) {
	recreateCoin := c.GetCoinBySymbol(symbol)
	if recreateCoin == nil {
		panic("coin to recreate does not exists")
	}

	// update version for recreating coin
	symbolCoins := c.getBySymbol(symbol)

	// TODO: change array to sorted array by version and get the last one
	lastVersion := uint16(0)
	for _, id := range symbolCoins {
		coin := c.GetCoin(id)
		if coin.Version() > lastVersion {
			lastVersion = coin.Version()
		}
	}

	recreateCoin.CVersion = lastVersion + 1
	c.setToMap(recreateCoin.id, recreateCoin)
	c.markDirty(recreateCoin.id)

	c.Create(newID, recreateCoin.Symbol(), "", volume, crr, reserve, maxSupply, nil)
}

func (c *Coins) ChangeOwner(symbol types.CoinSymbol, owner types.Address) {
	info := c.getSymbolInfo(symbol)
	info.SetOwnerAddress(&owner)

	coin := c.GetCoinBySymbol(symbol)
	coin.symbolInfo = info

	c.setToMap(coin.id, coin)
	c.markDirty(coin.ID())
}

func (c *Coins) get(id types.CoinID) *Model {
	if id.IsBaseCoin() {
		// TODO: refactor
		return &Model{
			id:      types.GetBaseCoinID(),
			CSymbol: types.GetBaseCoin(),
			info: &Info{
				Volume:  big.NewInt(0),
				Reserve: big.NewInt(0),
			},
		}
	}

	if coin := c.getFromMap(id); coin != nil {
		return coin
	}

	_, enc := c.iavl.Get(getCoinPath(id))
	if len(enc) == 0 {
		return nil
	}

	coin := &Model{}
	if err := rlp.DecodeBytes(enc, coin); err != nil {
		panic(fmt.Sprintf("failed to decode coin at %d: %s", id, err))
	}

	coin.id = id
	coin.markDirty = c.markDirty

	// load info
	_, enc = c.iavl.Get(getCoinInfoPath(id))
	if len(enc) != 0 {
		var info Info
		if err := rlp.DecodeBytes(enc, &info); err != nil {
			panic(fmt.Sprintf("failed to decode coin info %d: %s", id, err))
		}

		coin.info = &info
	}

	c.setToMap(id, coin)

	return coin
}

func (c *Coins) getSymbolInfo(symbol types.CoinSymbol) *SymbolInfo {
	info := &SymbolInfo{}

	_, enc := c.iavl.Get(getSymbolInfoPath(symbol))
	if len(enc) == 0 {
		return info
	}

	if err := rlp.DecodeBytes(enc, info); err != nil {
		panic(fmt.Sprintf("failed to decode coin symbol %s: %s", symbol.String(), err))
	}

	return info
}

func (c *Coins) getBySymbol(symbol types.CoinSymbol) []types.CoinID {
	var coins []types.CoinID

	_, enc := c.iavl.Get(getSymbolCoinsPath(symbol))
	if len(enc) == 0 {
		return coins
	}

	if err := rlp.DecodeBytes(enc, &coins); err != nil {
		panic(fmt.Sprintf("failed to decode coins by symbol %s: %s", symbol, err))
	}

	return coins
}

func (c *Coins) markDirty(id types.CoinID) {
	c.dirty[id] = struct{}{}
}

func (c *Coins) getOrderedDirtyCoins() []types.CoinID {
	keys := make([]types.CoinID, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	return keys
}

func (c *Coins) Export(state *types.AppState) {
	c.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix {
			if key[1] == symbolPrefix {
				return false
			}

			coinID := types.BytesToCoinID(key[1:])
			coin := c.get(coinID)

			owner := c.getSymbolInfo(coin.Symbol()).OwnerAddress()

			state.Coins = append(state.Coins, types.Coin{
				ID:           coin.ID(),
				Name:         coin.Name(),
				Symbol:       coin.Symbol(),
				Volume:       coin.Volume().String(),
				Crr:          coin.Crr(),
				Reserve:      coin.Reserve().String(),
				MaxSupply:    coin.MaxSupply().String(),
				Version:      coin.Version(),
				OwnerAddress: owner,
			})
		}

		return false
	})
}

func (c *Coins) getFromMap(id types.CoinID) *Model {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[id]
}

func (c *Coins) setToMap(id types.CoinID, model *Model) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[id] = model
}

func getSymbolCoinsPath(symbol types.CoinSymbol) []byte {
	path := append([]byte{mainPrefix}, []byte{symbolPrefix}...)
	return append(path, symbol.Bytes()...)
}

func getSymbolInfoPath(symbol types.CoinSymbol) []byte {
	path := append([]byte{mainPrefix}, []byte{symbolPrefix}...)
	path = append(path, symbol.Bytes()...)
	return append(path, []byte{infoPrefix}...)
}

func getCoinPath(id types.CoinID) []byte {
	return append([]byte{mainPrefix}, id.Bytes()...)
}

func getCoinInfoPath(id types.CoinID) []byte {
	return append(getCoinPath(id), infoPrefix)
}
