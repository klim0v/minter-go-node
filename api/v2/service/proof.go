package service

import (
	"bytes"
	"context"
	"encoding/hex"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testKey = func() (b []byte) { // todo
	b = []byte{'a'}
	a := types.HexToAddress("Mx")
	b = append(b, a[:]...)
	b = append(b, 'b')
	b = append(b, 0, 0, 0, 0)
	return b
}()

func (s *Service) Proof(ctx context.Context, req *pb.ProofRequest) (*pb.ProofResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	iTree := cState.ITree()
	rootHash := iTree.Hash()

	response := &pb.ProofResponse{
		StorageHash: "0x" + hex.EncodeToString(rootHash),
	}
	for i, hexKey := range req.Keys {
		key, err := hex.DecodeString(hexKey)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid key[%d]", i)
		}

		value, rangeProof, err := iTree.GetWithProof(key)

		var proof []string
		var hash []byte

		for i, leaf := range rangeProof.Leaves {
			hash = leaf.Hash()
			pl := rangeProof.LeftPath[:len(rangeProof.LeftPath)-i]
			for i := len(pl) - 1; i >= 0; i-- {
				pin := pl[i]
				hash = pin.Hash(hash)
				if bytes.Equal(hash, rootHash) {
					break
				}
				proof = append(proof, "0x"+hex.EncodeToString(hash))
			}
		}

		response.StorageProof = append(response.StorageProof,
			&pb.ProofResponse_StorageProof{
				Key:   "0x" + hex.EncodeToString(key),
				Proof: proof,
				Value: "0x" + hex.EncodeToString(value),
			})
	}

	return response, nil
}
