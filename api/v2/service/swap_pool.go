package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/helpers"

	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const precision = 34

func (s *Service) LimitOrdersOfPool(ctx context.Context, req *pb.LimitOrdersOfPoolRequest) (*pb.LimitOrdersOfPoolResponse, error) {
	if req.SellCoin == req.BuyCoin {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	swapper := cState.Swap().GetSwapper(types.CoinID(req.BuyCoin), types.CoinID(req.SellCoin))
	if swapper.GetID() == 0 {
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	limit := req.Limit
	if limit == 0 {
		limit = 999999
	}
	capacity := limit
	if capacity > 50 {
		capacity = 50
	}
	resp := &pb.LimitOrdersOfPoolResponse{Orders: make([]*pb.LimitOrderResponse, 0, capacity), PoolPrice: swapper.PriceRat().FloatString(precision)}

	orders := swapper.OrdersSell(uint32(limit))
	for _, order := range orders {
		if order == nil {
			break
		}
		resp.Orders = append(resp.Orders, &pb.LimitOrderResponse{
			Id: uint64(order.ID()),
			CoinSell: &pb.Coin{
				Id:     uint64(order.Coin1),
				Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
			},
			CoinBuy: &pb.Coin{
				Id:     uint64(order.Coin0),
				Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
			},
			WantSell: order.WantSell.String(),
			WantBuy:  order.WantBuy.String(),
			Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
			Owner:    order.Owner.String(),
			Height:   order.Height,
		})
	}

	return resp, nil
}

func (s *Service) LimitOrder(ctx context.Context, req *pb.LimitOrderRequest) (*pb.LimitOrderResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	order := cState.Swap().GetOrder(uint32(req.OrderId))
	if order == nil {
		return nil, status.Error(codes.NotFound, "limit order not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	if order.IsBuy {
		order = order.Reverse()
	}

	return &pb.LimitOrderResponse{
		Id: uint64(order.ID()),
		CoinSell: &pb.Coin{
			Id:     uint64(order.Coin1),
			Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
		},
		CoinBuy: &pb.Coin{
			Id:     uint64(order.Coin0),
			Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
		},
		WantSell: order.WantSell.String(),
		WantBuy:  order.WantBuy.String(),
		Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
		Owner:    order.Owner.String(),
		Height:   order.Height,
	}, nil
}

func (s *Service) LimitOrders(ctx context.Context, req *pb.LimitOrdersRequest) (*pb.LimitOrdersResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	capacity := len(req.Ids)
	if capacity > 1000 || capacity == 0 {
		capacity = 1000
	}

	resp := &pb.LimitOrdersResponse{Orders: make([]*pb.LimitOrderResponse, 0, capacity)}
	if capacity == 0 {
		orders := cState.Swap().GetOrdersAll(ctx)
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		for _, order := range orders {
			if order.IsBuy {
				order = order.Reverse()
			}

			resp.Orders = append(resp.Orders, &pb.LimitOrderResponse{
				Id: uint64(order.ID()),
				CoinSell: &pb.Coin{
					Id:     uint64(order.Coin1),
					Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
				},
				CoinBuy: &pb.Coin{
					Id:     uint64(order.Coin0),
					Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
				},
				WantSell: order.WantSell.String(),
				WantBuy:  order.WantBuy.String(),
				Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
				Owner:    order.Owner.String(),
				Height:   order.Height,
			})
		}

		return resp, nil
	}

	for _, id := range req.Ids {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		order := cState.Swap().GetOrder(uint32(id))
		if order == nil {
			continue
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		if order.IsBuy {
			order = order.Reverse()
		}

		resp.Orders = append(resp.Orders, &pb.LimitOrderResponse{
			Id: uint64(order.ID()),
			CoinSell: &pb.Coin{
				Id:     uint64(order.Coin1),
				Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
			},
			CoinBuy: &pb.Coin{
				Id:     uint64(order.Coin0),
				Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
			},
			WantSell: order.WantSell.String(),
			WantBuy:  order.WantBuy.String(),
			Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
			Owner:    order.Owner.String(),
			Height:   order.Height,
		})
	}

	return resp, nil
}

func (s *Service) LimitOrdersByOwner(ctx context.Context, req *pb.LimitOrdersByOwnerRequest) (*pb.LimitOrdersResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Address[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)

	resp := &pb.LimitOrdersResponse{Orders: make([]*pb.LimitOrderResponse, 0, 0)}

	orders := cState.Swap().GetOrdersByOwner(ctx, address)

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	for _, order := range orders {
		if order.IsBuy {
			order = order.Reverse()
		}

		resp.Orders = append(resp.Orders, &pb.LimitOrderResponse{
			Id: uint64(order.ID()),
			CoinSell: &pb.Coin{
				Id:     uint64(order.Coin1),
				Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
			},
			CoinBuy: &pb.Coin{
				Id:     uint64(order.Coin0),
				Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
			},
			WantSell: order.WantSell.String(),
			WantBuy:  order.WantBuy.String(),
			Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
			Owner:    order.Owner.String(),
			Height:   order.Height,
		})
	}

	return resp, nil
}

func (s *Service) SwapPool(ctx context.Context, req *pb.SwapPoolRequest) (*pb.SwapPoolResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	reserve0, reserve1, liquidityID := cState.Swap().SwapPool(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	if liquidityID == 0 {
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	return &pb.SwapPoolResponse{
		Id:        uint64(liquidityID),
		Price:     swap.CalcPriceSellRat(reserve1, reserve0).FloatString(precision),
		Amount0:   reserve0.String(),
		Amount1:   reserve1.String(),
		Liquidity: cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(liquidityID), 0).Volume().String(),
	}, nil
}

func (s *Service) SwapPoolProvider(ctx context.Context, req *pb.SwapPoolProviderRequest) (*pb.SwapPoolResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	if !strings.HasPrefix(strings.Title(req.Provider), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Provider[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}
	address := types.BytesToAddress(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	swapper := cState.Swap().GetSwapper(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	liquidityID := swapper.GetID()
	if liquidityID == 0 {
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	liquidityCoin := cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(liquidityID), 0)
	balance := cState.Accounts().GetBalance(address, liquidityCoin.ID())

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	amount0, amount1 := swapper.Amounts(balance, liquidityCoin.Volume())
	return &pb.SwapPoolResponse{
		Id:        uint64(swapper.GetID()),
		Price:     swapper.Reverse().PriceRat().FloatString(precision),
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
		Liquidity: balance.String(),
	}, nil
}

func (s *Service) SwapPools(ctx context.Context, req *pb.SwapPoolsRequest) (*pb.SwapPoolsResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	pools := cState.Swap().SwapPools(ctx)

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	sort.SliceStable(pools, func(i, j int) bool {
		return pools[i].GetID() < pools[j].GetID()
	})

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	res := &pb.SwapPoolsResponse{Pools: make([]*pb.SwapPoolsResponse_SwapPool, 0, len(pools))}
	for _, pool := range pools {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		var ordersSell []*pb.SwapPoolsResponse_SwapPool_LimitOrder
		var ordersBuy []*pb.SwapPoolsResponse_SwapPool_LimitOrder
		if req.Orders {
			for _, order := range pool.OrdersSell(uint32(10000)) {
				if order == nil {
					break
				}
				ordersSell = append(ordersSell, &pb.SwapPoolsResponse_SwapPool_LimitOrder{
					Id:       uint64(order.ID()),
					WantSell: order.WantSell.String(),
					WantBuy:  order.WantBuy.String(),
					Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
					Owner:    order.Owner.String(),
					Height:   order.Height,
				})
				if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
					return nil, timeoutStatus.Err()
				}
			}
			for _, order := range pool.Reverse().OrdersSell(uint32(10000)) {
				if order == nil {
					break
				}
				ordersBuy = append(ordersBuy, &pb.SwapPoolsResponse_SwapPool_LimitOrder{
					Id:       uint64(order.ID()),
					WantSell: order.WantSell.String(),
					WantBuy:  order.WantBuy.String(),
					Price:    swap.CalcPriceSellRat(order.WantBuy, order.WantSell).FloatString(precision),
					Owner:    order.Owner.String(),
					Height:   order.Height,
				})
				if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
					return nil, timeoutStatus.Err()
				}
			}
		}
		reserve0, reserve1 := pool.Reserves()
		res.Pools = append(res.Pools, &pb.SwapPoolsResponse_SwapPool{
			Id:         uint64(pool.GetID()),
			Price:      pool.Reverse().PriceRat().FloatString(precision),
			Coin0:      uint64(pool.Coin0()),
			Coin1:      uint64(pool.Coin1()),
			Amount0:    reserve0.String(),
			Amount1:    reserve1.String(),
			Liquidity:  cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(pool.GetID()), 0).Volume().String(),
			OrdersSell: ordersSell,
			OrdersBuy:  ordersBuy,
		})
	}

	return res, nil
}

func (s *Service) BestTrade(ctx context.Context, req *pb.BestTradeRequest) (*pb.BestTradeResponse, error) {
	amount := helpers.StringToBigIntOrNil(req.Amount)
	if amount == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("cannot decode %s into big.Int", amount).Error())
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	depth := req.MaxDepth
	if depth == 0 || depth > 4 {
		depth = 4
	}

	var trade *swap.Trade
	if req.Type == pb.BestTradeRequest_input {
		trade = cState.Swap().GetBestTradeExactIn(ctx, req.BuyCoin, req.SellCoin, amount, depth)
	} else {
		trade = cState.Swap().GetBestTradeExactOut(ctx, req.SellCoin, req.BuyCoin, amount, depth)
	}
	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	if trade == nil {
		return nil, status.Error(codes.NotFound, "route path not found")
	}

	route := &pb.BestTradeResponse{
		Path: make([]uint64, 0, len(trade.Route.Path)),
	}
	if req.Type == pb.BestTradeRequest_input {
		route.Result = trade.OutputAmount.Amount.String()
	} else {
		route.Result = trade.InputAmount.Amount.String()
	}
	for _, token := range trade.Route.Path {
		route.Path = append(route.Path, uint64(token))
	}

	return route, nil
}
