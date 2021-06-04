package swap

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/big"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
)

const commissionOrder = 2

func (s *Swap) PairSellWithOrders(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32, *ChangeDetailsWithOrders, map[types.Address]*big.Int) {
	pair := s.Pair(coin0, coin1)
	amount1Out, owners, details := pair.SellWithOrders(amount0In)
	if amount1Out.Cmp(minAmount1Out) == -1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", amount1Out, minAmount1Out))
	}

	for _, b := range owners {
		s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(b))
	}
	s.bus.Checker().AddCoin(coin0, amount0In)
	s.bus.Checker().AddCoin(coin1, big.NewInt(0).Neg(amount1Out))
	return amount0In, amount1Out, pair.GetID(), details, owners
}

func (s *Swap) PairBuyWithOrders(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32, *ChangeDetailsWithOrders, map[types.Address]*big.Int) {
	pair := s.Pair(coin0, coin1)
	amount0In, owners, details := pair.BuyWithOrders(amount1Out)
	if amount1Out.Cmp(maxAmount0In) == 1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", amount1Out, maxAmount0In))
	}

	for _, b := range owners {
		s.bus.Checker().AddCoin(coin0, big.NewInt(0).Neg(b))
	}
	s.bus.Checker().AddCoin(coin0, amount0In)
	s.bus.Checker().AddCoin(coin1, big.NewInt(0).Neg(amount1Out))
	return amount0In, amount1Out, pair.GetID(), details, owners
}

type ChangeDetailsWithOrders struct {
	SwapAmount0In           *big.Int `json:"swap_amount_0_in"`
	SwapAmount1Out          *big.Int `json:"swap_amount_1_out"`
	OrdersCommissionAmount0 *big.Int `json:"orders_commission_amount_0"`
	OrdersCommissionAmount1 *big.Int `json:"orders_commission_amount_1"`
	Orders                  []*Limit `json:"orders"`
}

func (p *Pair) SellWithOrders(amount0In *big.Int) (amount1Out *big.Int, owners map[types.Address]*big.Int, c *ChangeDetailsWithOrders) { // todo: add mutex
	if amount0In == nil || amount0In.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}
	amount1Out, orders := p.calculateBuyForSellWithOrders(amount0In)
	if amount1Out == nil || amount1Out.Sign() != 1 {
		panic(ErrorInsufficientOutputAmount)
	}

	commission0orders, commission1orders, amount0, amount1, owners := CalcDiffPool(amount0In, amount1Out, orders)

	log.Println("uS", commission0orders, commission1orders)

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		log.Println("a", amount0, amount1)
		p.update(amount0, big.NewInt(0).Neg(amount1))
	}

	p.update(commission0orders, commission1orders)

	p.updateOrders(orders)

	return amount1Out, owners, &ChangeDetailsWithOrders{
		SwapAmount0In:           amount0,
		SwapAmount1Out:          amount1,
		OrdersCommissionAmount0: commission0orders,
		OrdersCommissionAmount1: commission1orders,
		Orders:                  orders,
	}
}

func CalcDiffPool(amount0In, amount1Out *big.Int, orders []*Limit) (*big.Int, *big.Int, *big.Int, *big.Int, map[types.Address]*big.Int) {
	owners := map[types.Address]*big.Int{}

	amount0orders, amount1orders := big.NewInt(0), big.NewInt(0)
	commission0orders, commission1orders := big.NewInt(0), big.NewInt(0)
	for _, order := range orders {
		amount0orders.Add(amount0orders, order.WantBuy)
		amount1orders.Add(amount1orders, order.WantSell)

		cB := calcCommission999(order.WantBuy)
		cS := calcCommission999(order.WantSell)

		commission0orders.Add(commission0orders, cB)
		commission1orders.Add(commission1orders, cS)

		if owners[order.Owner] == nil {
			owners[order.Owner] = big.NewInt(0)
		}
		owners[order.Owner].Add(owners[order.Owner], big.NewInt(0).Sub(order.WantBuy, cB))
	}

	amount1orders.Sub(amount1orders, commission1orders)

	amount0 := big.NewInt(0).Sub(amount0In, amount0orders)
	amount1 := big.NewInt(0).Sub(amount1Out, amount1orders)

	return commission0orders, commission1orders, amount0, amount1, owners
}

func (p *Pair) BuyWithOrders(amount1Out *big.Int) (amount0In *big.Int, owners map[types.Address]*big.Int, c *ChangeDetailsWithOrders) { // todo: add mutex
	if amount1Out == nil || amount1Out.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}
	amount0In, orders := p.calculateSellForBuyWithOrders(amount1Out)
	if amount0In == nil || amount0In.Sign() != 1 {
		panic(ErrorInsufficientOutputAmount)
	}

	commission0orders, commission1orders, amount0, amount1, owners := CalcDiffPool(amount0In, amount1Out, orders)

	log.Println("uB", commission0orders, commission1orders)

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		log.Println("a", amount0, amount1)
		p.update(amount0, big.NewInt(0).Neg(amount1))
	}

	p.update(commission0orders, commission1orders)

	p.updateOrders(orders)

	return amount0In, owners, &ChangeDetailsWithOrders{
		SwapAmount0In:           amount0,
		SwapAmount1Out:          amount1,
		OrdersCommissionAmount0: commission0orders,
		OrdersCommissionAmount1: commission1orders,
		Orders:                  orders,
	}
}

func (p *Pair) updateOrders(orders []*Limit) {
	var editedOrders []*Limit
	for i, order := range orders {
		editedOrders = append(editedOrders, p.updateSellLowerOrder(i, order.WantBuy, order.WantSell))
	}
	for _, editedOrder := range editedOrders {
		p.resortSellOrderList(0, editedOrder)
	}

	p.markDirtyOrders()
}

func (p *Pair) updateSellLowerOrder(i int, amount0, amount1 *big.Int) *Limit {
	limit := p.orderSellLowerByIndex(i)

	newLimit := limit.sort()
	newLimit.OldSortPrice()

	limit.WantBuy.Sub(limit.WantBuy, amount0)
	limit.WantSell.Sub(limit.WantSell, amount1)

	p.MarkDirtyOrders(newLimit) // need before resort

	return newLimit
}

func (p *Pair) resortSellOrderList(i int, limit *Limit) {
	if limit.isEmpty() {
		if !(limit.WantBuy.Sign() == 0 && limit.WantSell.Sign() == 0) {
			panic(fmt.Sprintf("zero value of %#v", limit))
		}
		p.unsetOrderSellLowerByIndex(i)
		return
	}

	cmp := 1
	if !p.isSorted() {
		cmp = -1
	}
	switch limit.CmpOldRate() {
	case 0:
		return
	case cmp:
		p.unsetOrderSellLowerByIndex(i)
	default:
		p.unsetOrderSellLowerByIndex(i)

		loadedLen := len(p.SellLowerOrders())
		newIndex := p.setSellLowerOrder(limit)
		if newIndex == loadedLen {
			p.unsetOrderSellLowerByIndex(newIndex)
			p.setOrder(limit)
		}
	}
}

func (l *Limit) isEmpty() bool {
	return l.WantBuy.Sign() == 0 || l.WantSell.Sign() == 0
}

func (l *Limit) isKeepRate() bool {
	return l.CmpOldRate() == 0
}

func (l *Limit) CmpOldRate() int {
	return l.SortPrice().Cmp(l.OldSortPrice())
}

func (p *Pair) CalculateBuyForSellWithOrders(amount0In *big.Int) (amount1Out *big.Int) {
	amount1Out, _ = p.calculateBuyForSellWithOrders(amount0In)
	return amount1Out
}

func (p *Pair) calculateBuyForSellWithOrders(amount0In *big.Int) (amountOut *big.Int, orders []*Limit) {
	amountOut = big.NewInt(0)
	amountIn := big.NewInt(0).Set(amount0In)
	var pair EditableChecker = p
	for i := 0; true; i++ {
		if amountIn.Sign() == 0 {
			return amountOut, orders
		}

		limit := p.orderSellLowerByIndex(i)
		if limit == nil {
			break
		}

		price := limit.Price()
		if price.Cmp(pair.Price()) == -1 {
			reserve0diff := pair.CalculateAddAmount0ForPrice(price)
			if reserve0diff != nil {
				if amountIn.Cmp(reserve0diff) != 1 {
					break
				}

				reserve1diff := pair.CalculateBuyForSell(reserve0diff)
				if reserve1diff == nil {
					reserve1diff = big.NewInt(0) // todo: mb break `if`
				}

				amountIn.Sub(amountIn, reserve0diff)
				amountOut.Add(amountOut, reserve1diff)

				if err := pair.CheckSwap(reserve0diff, reserve1diff); err != nil {
					panic(err) // todo: for test
				}
				pair = pair.AddLastSwapStep(reserve0diff, reserve1diff)

				log.Println("rS", reserve0diff, reserve1diff)
			}
		}

		// хотим продать 9009
		// проверяем есть ли столько на продажу
		log.Println("i", amountIn)
		rest := big.NewInt(0).Sub(amountIn, limit.WantBuy)
		if rest.Sign() != 1 {
			// 9009
			amount0 := big.NewInt(0).Set(amountIn)
			// считаем сколько сможем купить -- 3003
			amount1, acc := big.NewFloat(0).Mul(price, big.NewFloat(0).SetInt(amount0)).Int(nil)
			if acc != big.Exact {
				log.Println("acc", acc)
				// if acc == big.Below { // todo
				// 	amount1.Add(amount1,big.NewInt(1))
				// }
			}
			log.Println("m", amount1)

			orders = append(orders, &Limit{
				isBuy:        limit.isBuy,
				pairKey:      p.pairKey,
				WantBuy:      amount0, // 9009, 9 заберем в пул
				WantSell:     amount1, // 3003, 3 пул
				Owner:        limit.Owner,
				oldSortPrice: limit.SortPrice(),
				id:           limit.id,
				RWMutex:      new(sync.RWMutex),
			})

			comB := calcCommission999(amount1)
			log.Println("n", comB)
			amountOut.Add(amountOut, big.NewInt(0).Sub(amount1, comB))
			return amountOut, orders
		}

		orders = append(orders, &Limit{
			isBuy:        limit.isBuy,
			WantBuy:      big.NewInt(0).Set(limit.WantBuy),
			WantSell:     big.NewInt(0).Set(limit.WantSell),
			Owner:        limit.Owner,
			pairKey:      limit.pairKey,
			oldSortPrice: limit.SortPrice(),
			id:           limit.id,
			RWMutex:      limit.RWMutex,
		})

		comS := calcCommission999(limit.WantBuy)
		comB := calcCommission999(limit.WantSell)

		pair = pair.AddLastSwapStep(comS, big.NewInt(0).Neg(comB))
		amountOut.Add(amountOut, big.NewInt(0).Sub(limit.WantSell, comB))

		amountIn = rest
	}

	amount1diff := pair.CalculateBuyForSell(amountIn)
	if amount1diff != nil {
		if err := pair.CheckSwap(amountIn, amount1diff); err != nil {
			panic(err) // todo: for test
		}
		amountOut.Add(amountOut, amount1diff)
	}
	return amountOut, orders
}

func calcCommission001(amount0 *big.Int) *big.Int {
	mul := big.NewInt(0).Mul(amount0, big.NewInt(commissionOrder/2))
	quo := big.NewInt(0).Quo(mul, big.NewInt(1000))
	remainder := big.NewInt(0)
	if big.NewInt(0).Rem(mul, big.NewInt(1000)).Sign() == 1 {
		remainder = big.NewInt(1)
	}
	quo.Add(quo, remainder)
	return quo
}

func calcCommission999(amount1 *big.Int) *big.Int {
	quo := big.NewInt(0).Quo(amount1, big.NewInt(1000+commissionOrder/2))
	remainder := big.NewInt(0)
	if big.NewInt(0).Rem(amount1, big.NewInt(1000+commissionOrder/2)).Sign() == 1 {
		remainder = big.NewInt(1)
	}
	quo.Add(quo, remainder)
	return quo
}

func (p *Pair) CalculateAddAmount0ForPrice(price *big.Float) (amount0 *big.Int) {
	if price.Cmp(p.Price()) == 1 {
		amount0 := p.reverse().CalculateAddAmount1ForPrice(big.NewFloat(0).Quo(big.NewFloat(1), price))
		return amount0.Neg(amount0)
	}
	return p.calculateAddAmount0ForPrice(price)
}

// Deprecated
func (p *Pair) CalculateAddAmount1ForPrice(price *big.Float) (amount1 *big.Int) {
	if price.Cmp(p.Price()) == 1 {
		amount1 := p.reverse().CalculateAddAmount0ForPrice(big.NewFloat(0).Quo(big.NewFloat(1), price))
		return amount1.Neg(amount1)
	}
	return p.calculateAddAmount1ForPrice(price)
}

// Deprecated
func (p *Pair) calculateAddAmount1ForPrice(price *big.Float) (amount1 *big.Int) {
	amount0 := p.calculateAddAmount0ForPrice(price)
	return p.CalculateBuyForSellAllowNeg(amount0)
}

func (p *Pair) calculateAddAmount0ForPrice(price *big.Float) (amount0 *big.Int) { // todo: mb Div by ZERO???
	reserve0, reserve1 := p.Reserves()
	r0 := big.NewFloat(0).SetInt(reserve0)
	r1 := big.NewFloat(0).SetInt(reserve1)
	k := big.NewFloat(0).Mul(r0, r1)

	a := big.NewFloat((1000 + commission) / 1000)
	b := big.NewFloat(0).Quo(big.NewFloat(0).Mul(big.NewFloat(2000+commission), r0), big.NewFloat(1000))
	c := big.NewFloat(0).Sub(big.NewFloat(0).Mul(r0, r0), big.NewFloat(0).Quo(k, price))
	d := big.NewFloat(0).Sub(big.NewFloat(0).Mul(b, b), big.NewFloat(0).Mul(big.NewFloat(4), big.NewFloat(0).Mul(a, c)))

	x := big.NewFloat(0).Quo(big.NewFloat(0).Add(big.NewFloat(0).Neg(b), big.NewFloat(0).Sqrt(d)), big.NewFloat(0).Mul(big.NewFloat(2), a))

	amount0, _ = big.NewFloat(0).Add(x, big.NewFloat(0).Quo(big.NewFloat(0).Mul(big.NewFloat(2), x), big.NewFloat(1000))).Int(nil)

	return amount0
	// return amount0.Add(amount0, big.NewInt(1))
}

func (p *Pair) CalculateSellForBuyWithOrders(amount1Out *big.Int) (amount0In *big.Int) {
	amount0In, _ = p.calculateSellForBuyWithOrders(amount1Out)
	return amount0In
}

func (p *Pair) calculateSellForBuyWithOrders(amount1Out *big.Int) (amountIn *big.Int, orders []*Limit) {
	amountIn = big.NewInt(0)
	amountOut := big.NewInt(0).Set(amount1Out)
	var pair EditableChecker = p
	for i := 0; true; i++ {
		if amountOut.Sign() == 0 {
			return amountIn, orders
		}

		limit := p.orderSellLowerByIndex(i)
		if limit == nil {
			break
		}
		log.Println("ow", limit.Owner.String())

		price := limit.Price()
		if price.Cmp(pair.Price()) == -1 {
			reserve0diff := pair.CalculateAddAmount0ForPrice(price)
			if reserve0diff != nil {
				reserve1diff := pair.CalculateBuyForSell(reserve0diff)
				if reserve1diff == nil {
					reserve1diff = big.NewInt(0) // todo: mb break `if`
				}
				if amountOut.Cmp(reserve1diff) != 1 {
					break
				}

				amountOut.Sub(amountOut, reserve1diff)
				amountIn.Add(amountIn, reserve0diff)

				if err := pair.CheckSwap(reserve0diff, reserve1diff); err != nil {
					panic(err) // todo: for test
				}
				pair = pair.AddLastSwapStep(reserve0diff, reserve1diff)

				log.Println("rB", reserve0diff, reserve1diff)
			}
		}

		// хочу купить amountOut = 3000
		// тк мы 0.1% положим в пул, то надо купить 3003
		// проверим что в пуле есть X * 0.999 == 3000
		// на продажу есть 5000
		// что бы в пул пошел 0.1%, мне надо купить 3003 из которых 3 положить в пул
		comB := calcCommission999(limit.WantSell)
		log.Println("amountOut", amountOut)
		log.Println("comB", comB)
		rest := big.NewInt(0).Sub(amountOut, big.NewInt(0).Sub(limit.WantSell, comB))
		if rest.Sign() != 1 {
			log.Println("part")
			amount1 := big.NewInt(0).Add(amountOut, calcCommission001(amountOut))
			log.Println("amount1", amount1)
			// считаем сколько монет надо продать что бы купить 3003
			amount0, acc := big.NewFloat(0).Quo(big.NewFloat(0).SetInt(amount1), price).Int(nil)
			if acc != big.Exact {
				log.Println("acc", acc) // todo
			}
			log.Println("amount0", amount0)

			// 7330916069244652544 4225079013582808273
			// 7330916069244652544 4225079013582808273

			orders = append(orders, &Limit{
				isBuy:        limit.isBuy,
				pairKey:      p.pairKey,
				WantBuy:      big.NewInt(0).Set(amount0), // и того продам по ордеру 9009, из них 9000 продавцу и 9 в пул
				WantSell:     amount1,                    // 3003, позже вычтем 3 и положим в пул
				Owner:        limit.Owner,
				oldSortPrice: limit.SortPrice(),
				id:           limit.id,
				RWMutex:      new(sync.RWMutex),
			})

			amountIn.Add(amountIn, amount0)
			return amountIn, orders
		}

		log.Println("full")
		orders = append(orders, &Limit{
			isBuy:        limit.isBuy,
			WantBuy:      big.NewInt(0).Set(limit.WantBuy),
			WantSell:     big.NewInt(0).Set(limit.WantSell),
			Owner:        limit.Owner,
			pairKey:      limit.pairKey,
			oldSortPrice: limit.SortPrice(),
			id:           limit.id,
			RWMutex:      limit.RWMutex,
		})

		comS := calcCommission999(limit.WantBuy)

		pair = pair.AddLastSwapStep(comS, big.NewInt(0).Neg(comB))
		amountOut = rest

		amountIn.Add(amountIn, limit.WantBuy)
		// amountIn.Add(amountIn, comS)
	}

	amount0diff := pair.CalculateSellForBuy(amountOut)
	if amount0diff != nil {
		if err := pair.CheckSwap(amount0diff, amountOut); err != nil {
			panic(err) // todo: for test
		}
		amountIn.Add(amountIn, amount0diff)
	}
	return amountIn, orders
}

func CalcPriceSell(sell, buy *big.Int) *big.Float {
	return new(big.Float).SetPrec(precision).Quo(
		big.NewFloat(0).SetInt(buy),
		big.NewFloat(0).SetInt(sell),
	)
}

type Limit struct {
	isBuy    bool
	WantBuy  *big.Int `json:"buy"`
	WantSell *big.Int `json:"sell"`

	Owner types.Address `json:"owner"`

	pairKey
	oldSortPrice *big.Float
	id           uint32

	*sync.RWMutex
}

type limits struct {
	higher []*Limit
	lower  []*Limit
	// todo: add mutex
}

type dirtyOrders struct {
	orders map[uint32]*Limit // list sorted dirties Order
}

const (
	precision = 54 // supported precision
)

func (l *Limit) Price() *big.Float {
	if l.isEmpty() {
		return big.NewFloat(0)
	}
	l.RLock()
	defer l.RUnlock()
	return CalcPriceSell(l.WantBuy, l.WantSell)
}

func (l *Limit) SortPrice() *big.Float {
	if l.isSorted() {
		return l.Price()
	}
	return l.reverse().Price()
}

func (l *Limit) OldSortPrice() *big.Float {
	if l.oldSortPrice == nil {
		l.oldSortPrice = l.SortPrice()
	}
	return l.oldSortPrice
}

func (l *Limit) isSell() bool {
	return !l.isBuy
}

// ReCalcOldSortPrice saves before change, need for update on disk
func (l *Limit) ReCalcOldSortPrice() *big.Float {
	l.oldSortPrice = l.SortPrice()
	return l.oldSortPrice
}

func (l *Limit) reverse() *Limit {
	l.RLock()
	defer l.RUnlock()
	return &Limit{
		pairKey:      l.pairKey.reverse(),
		isBuy:        !l.isBuy,
		WantBuy:      l.WantSell,
		WantSell:     l.WantBuy,
		Owner:        l.Owner,
		oldSortPrice: l.oldSortPrice,
		id:           l.id,
		RWMutex:      l.RWMutex,
	}
}

func (l *Limit) sort() *Limit {
	if l.isSorted() {
		return l
	}

	return l.reverse()
}

func (l *Limit) isSorted() bool {
	return l.pairKey.isSorted()
}

func (p *Pair) MarkDirtyOrders(order *Limit) {
	p.markDirtyOrders()
	p.dirtyOrders.orders[order.id] = order
	return
}

func (p *Pair) setSellHigherOrder(new *Limit) (index int) {
	cmp := -1
	if !p.isSorted() {
		cmp = 1
	}
	orders := p.sellHigherOrders()
	for i, limit := range orders {
		if new.SortPrice().Cmp(limit.SortPrice()) != cmp {
			index = i + 1
			continue
		}
		break
	}

	if index == 0 {
		p.setSellHigherOrders(append([]*Limit{new}, orders...))
		return
	}

	if index == len(orders) {
		p.setSellHigherOrders(append(orders, new))
		return
	}

	p.setSellHigherOrders(append(orders[:index], append([]*Limit{new}, orders[index:]...)...))
	return
}

func (p *Pair) setSellLowerOrder(new *Limit) (index int) {
	cmp := -1
	if p.isSorted() {
		cmp = 1
	}

	orders := p.SellLowerOrders()
	for i, limit := range orders {
		if new.SortPrice().Cmp(limit.SortPrice()) != cmp {
			index = i + 1
			continue
		}
		break
	}

	if index == 0 {
		p.setSellLowerOrders(append([]*Limit{new}, orders...))
		return
	}

	if index == len(orders) {
		p.setSellLowerOrders(append(orders, new))
		return
	}

	p.setSellLowerOrders(append(orders[:index], append([]*Limit{new}, orders[index:]...)...))
	return
}

func (p *Pair) SellLowerOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.lower
	}
	return p.buyOrders.higher
}

func (p *Pair) sellHigherOrders() []*Limit {
	if p.isSorted() {
		return p.sellOrders.higher
	}
	return p.buyOrders.lower
}
func (p *Pair) BuyHigherOrders() []*Limit {
	if p.isSorted() {
		return p.buyOrders.higher
	}
	return p.sellOrders.lower
}
func (p *Pair) buyLowerOrders() []*Limit {
	if p.isSorted() {
		return p.buyOrders.lower
	}
	return p.sellOrders.higher
}

func (p *Pair) setSellLowerOrders(orders []*Limit) {
	if p.isSorted() {
		p.sellOrders.lower = orders
		return
	}
	p.buyOrders.higher = orders
	return
}
func (p *Pair) setSellHigherOrders(orders []*Limit) {
	if p.isSorted() {
		p.sellOrders.higher = orders
		return
	}
	p.buyOrders.lower = orders
	return
}
func (p *Pair) setBuyHigherOrders(orders []*Limit) {
	if p.isSorted() {
		p.buyOrders.higher = orders
		return
	}
	p.sellOrders.lower = orders
	return
}
func (p *Pair) setBuyLowerOrders(orders []*Limit) {
	if p.isSorted() {
		p.buyOrders.lower = orders
		return
	}
	p.sellOrders.higher = orders
	return
}

func (s *Swap) PairAddOrder(coinWantBuy, coinWantSell types.CoinID, wantBuyAmount, wantSellAmount *big.Int, sender types.Address) uint32 {
	pair := s.Pair(coinWantBuy, coinWantSell)
	order := pair.SetOrder(wantBuyAmount, wantSellAmount, sender)

	s.bus.Checker().AddCoin(coinWantSell, wantSellAmount)

	return order.id
}

func (p *Pair) SetOrder(wantBuyAmount0, wantSellAmount1 *big.Int, sender types.Address) (order *Limit) {
	order = &Limit{
		pairKey:  p.pairKey,
		isBuy:    false,
		WantBuy:  wantBuyAmount0,
		WantSell: wantSellAmount1,
		id:       p.getLastTotalOrderID(),
		Owner:    sender,
		RWMutex:  new(sync.RWMutex),
	}
	defer p.MarkDirtyOrders(order.sort())

	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()

	p.setOrder(order)

	return order
}

func (p *Pair) setOrder(limit *Limit) {
	if p.Price().Cmp(limit.Price()) == -1 {
		// todo: do not allow
		log.Println("do not allow")
		p.setSellHigherOrder(limit.sort())
	} else {
		p.setSellLowerOrder(limit.sort())
	}
}

func (p *Pair) loadAllOrders(immutableTree *iavl.ImmutableTree) (orders []*Limit) {
	const countFirstBytes = 10

	startKey := append(append([]byte{mainPrefix}, p.pathOrders()...), byte(0), byte(0))
	endKey := append(append([]byte{mainPrefix}, p.pathOrders()...), byte(1), byte(255)) // todo: mb more high bytes

	immutableTree.IterateRange(startKey, endKey, true, func(key []byte, value []byte) bool {
		var isSell = true
		if key[countFirstBytes : countFirstBytes+1][0] == 0 {
			isSell = false
		}
		order := &Limit{
			id:      binary.BigEndian.Uint32(key[len(key)-4:]),
			pairKey: p.pairKey.sort(),
			isBuy:   !isSell,
			RWMutex: new(sync.RWMutex),
		}
		err := rlp.DecodeBytes(value, order)
		if err != nil {
			panic(err)
		}

		orders = append(orders, order)

		return false
	})

	return orders
}

// loadBuyHigherOrders loads only needed orders for pair, not all
func (s *Swap) loadBuyHigherOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex
	endKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(0), byte(255)) // todo: mb more high bytes
	var startKey []byte

	sliceLen := len(slice)
	if sliceLen > 0 {
		var l = slice[sliceLen-1]
		startKey = pricePath(pair.pairKey, l.SortPrice(), l.id+1, false)
	} else {
		startKey = pricePath(pair.pairKey, pair.SortPrice(), 0, false)
	}

	i := sliceLen
	s.immutableTree().IterateRange(startKey, endKey, true, func(key []byte, value []byte) bool {
		if i > limit {
			return true
		}

		order := &Limit{
			id:      binary.BigEndian.Uint32(key[len(key)-4:]),
			pairKey: pair.pairKey.sort(),
			isBuy:   true,
			RWMutex: new(sync.RWMutex),
		}
		err := rlp.DecodeBytes(value, order)
		if err != nil {
			panic(err)
		}

		if dirtyOrder, ok := pair.dirtyOrders.orders[order.id]; ok {
			if dirtyOrder.isKeepRate() {
				order = dirtyOrder
			} else if dirtyOrder.isEmpty() {
				return false
			} else {
				return false
			}
		}

		slice = append(slice, order)
		i++
		return false
	})

	return slice
}

func (s *Swap) loadSellLowerOrders(pair *Pair, slice []*Limit, limit int) []*Limit { // todo: add mutex
	startKey := append(append([]byte{mainPrefix}, pair.pathOrders()...), byte(1), byte(0))
	var endKey []byte

	sliceLen := len(slice)
	if sliceLen > 0 {
		var l = slice[sliceLen-1]
		endKey = pricePath(pair.pairKey, l.SortPrice(), l.id-1, true)
	} else {
		endKey = pricePath(pair.pairKey, pair.SortPrice(), math.MaxInt32, true)
	}

	i := sliceLen
	s.immutableTree().IterateRange(startKey, endKey, false, func(key []byte, value []byte) bool {
		if i > limit {
			return true
		}

		order := &Limit{
			id:      binary.BigEndian.Uint32(key[len(key)-4:]),
			pairKey: pair.pairKey.sort(),
			isBuy:   false,
			RWMutex: new(sync.RWMutex),
		}
		err := rlp.DecodeBytes(value, order)
		if err != nil {
			panic(err)
		}

		if dirtyOrder, ok := pair.dirtyOrders.orders[order.id]; ok {
			if dirtyOrder.isKeepRate() {
				order = dirtyOrder
			} else if dirtyOrder.isEmpty() {
				return false
			} else {
				return false
			}
		}

		slice = append(slice, order)
		i++
		return false
	})

	return slice
}

func (p *Pair) updateDirtyOrders(list []*Limit, lower bool) (orders []*Limit, countDirties int) {
	for _, order := range list {
		if dirtyOrder, ok := p.dirtyOrders.orders[order.id]; ok {
			if dirtyOrder.isKeepRate() {
				orders = append(orders, order)
				continue
			} else {
				countDirties++
				continue
			}
		}
		orders = append(orders, order)
	}

	cmp := 1
	if !p.isSorted() && lower || p.isSorted() && !lower {
		cmp *= -1
	}
	for _, dirtyOrder := range p.getDirtyOrdersList() {
		if dirtyOrder.isKeepRate() {
			continue
		}
		if dirtyOrder.isEmpty() {
			continue
		} else {
			var isSet bool
			orders, isSet = addToList(orders, dirtyOrder, cmp)
			if isSet {
				countDirties--
			}
		}
	}
	return orders, countDirties
}

func addToList(orders []*Limit, dirtyOrder *Limit, cmp int) (list []*Limit, includedInInterval bool) {
	var index int
	for i, limit := range orders {
		if dirtyOrder.SortPrice().Cmp(limit.SortPrice()) == cmp {
			index = i + 1
			continue
		}
		break
	}

	if index == 0 {
		return append([]*Limit{dirtyOrder}, orders...), true
	}

	if index == len(orders) {
		// not add to end
		return orders, false
		// return append(orders, dirtyOrder), true
	}

	return append(orders[:index], append([]*Limit{dirtyOrder}, orders[index:]...)...), true
}

func (p *Pair) OrderBuyHigherByIndex(index int) *Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderBuyHigherByIndex(index)
}

func (p *Pair) orderBuyHigherByIndex(index int) *Limit {
	orders := p.BuyHigherOrders()
	var count int
	var deleteCount int
	for firstIterate := true; (firstIterate && len(orders) <= index) || deleteCount != 0; firstIterate = false {
		orders, deleteCount = p.updateDirtyOrders(p.loadHigherOrders(p, orders, index+count), false)
		count += deleteCount
	}

	p.setBuyHigherOrders(orders)

	if len(orders)-1 < index {
		return nil
	}
	order := orders[index]
	if !p.isSorted() {
		return order.reverse()
	}

	return order
}

func (p *Pair) OrderBuyHigherLast() (limit *Limit, index int) {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderBuyHigherLast()
}

func (p *Pair) orderBuyHigherLast() (limit *Limit, index int) {
	for order := p.orderBuyHigherByIndex(index); order != nil; order = p.orderBuyHigherByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}

func (p *Pair) unsetOrderBuyHigherByIndex(index int) {
	slice := p.BuyHigherOrders()
	length := len(slice)

	if length <= index {
		panic(fmt.Sprintf("slice len %d, want index %d", length, index))
	}

	if length == 1 {
		p.setBuyHigherOrders(nil)
		return
	}

	switch index {
	case 0:
		slice = slice[index+1:]
	case length - 1:
		slice = slice[:index]
	default:
		slice = append(slice[:index], slice[index+1:]...)
	}

	p.setBuyHigherOrders(slice)
	return
}

func (p *Pair) unsetOrderSellLowerByIndex(index int) {
	slice := p.SellLowerOrders()
	length := len(slice)

	if length <= index {
		panic(fmt.Sprintf("slice len %d, want index %d", length, index))
	}

	if length == 1 {
		p.setSellLowerOrders(nil)
		return
	}

	switch index {
	case 0:
		slice = slice[index+1:]
	case length - 1:
		slice = slice[:index]
	default:
		slice = append(slice[:index], slice[index+1:]...)
	}

	p.setSellLowerOrders(slice)
	return
}

func (p *Pair) OrderSellLowerByIndex(index int) *Limit {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderSellLowerByIndex(index)
}
func (p *Pair) orderSellLowerByIndex(index int) *Limit {
	orders := p.SellLowerOrders()
	var count int
	var deleteCount int
	for firstIterate := true; (firstIterate && len(orders) <= index) || deleteCount != 0; firstIterate = false {
		orders, deleteCount = p.updateDirtyOrders(p.loadLowerOrders(p, orders, index+count), true)
		count += deleteCount
	}

	p.setSellLowerOrders(orders)

	if len(orders)-1 < index {
		return nil
	}

	order := orders[index]
	log.Println(order.Owner.String())
	if !p.isSorted() {
		return order.reverse()
	}

	return order
}

func (p *Pair) OrderSellLowerLast() (limit *Limit, index int) {
	p.lockOrders.Lock()
	defer p.lockOrders.Unlock()
	return p.orderSellLowerLast()
}
func (p *Pair) orderSellLowerLast() (limit *Limit, index int) {
	for order := p.orderSellLowerByIndex(index); order != nil; order = p.orderSellLowerByIndex(index) {
		limit = order
		index++
	}
	return limit, index - 1
}