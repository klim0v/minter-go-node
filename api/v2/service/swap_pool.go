package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"strings"

	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	resp := &pb.LimitOrdersOfPoolResponse{Orders: make([]*pb.LimitOrderResponse, 0, capacity)}

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
			Price:    swap.CalcPriceSell(order.WantBuy, order.WantSell).Text('f', 18),
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
		Price:    swap.CalcPriceSell(order.WantBuy, order.WantSell).Text('f', 18),
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
	if capacity > 50 {
		capacity = 50
	}
	resp := &pb.LimitOrdersResponse{Orders: make([]*pb.LimitOrderResponse, 0, capacity)}

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
			Price:    swap.CalcPriceSell(order.WantBuy, order.WantSell).Text('f', 18),
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
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
		Liquidity: balance.String(),
	}, nil
}