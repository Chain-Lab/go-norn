/**
  @author: decision
  @date: 2023/11/28
  @note:
**/

package rpc

import (
	"context"
	"encoding/hex"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/node"
	"github.com/chain-lab/go-chronos/rpc/pb"
	"github.com/chain-lab/go-chronos/utils"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type blockchainService struct {
	pb.UnimplementedBlockchainServer
	// todo: block cache?
}

func (s *blockchainService) GetBlockNumber(ctx context.Context,
	in *emptypb.Empty) (resp *pb.BlockNumberResp, err error) {
	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()

	resp = new(pb.BlockNumberResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Number = proto.Uint64(uint64(chain.Height()))

	return resp, nil
}

func (s *blockchainService) GetBlockByHash(ctx context.Context,
	in *pb.GetBlockReq) (resp *pb.GetBlockResp, err error) {
	// todo: validate the hash value
	hash := in.Hash
	full := in.Full
	byteHash, err := hex.DecodeString(*hash)

	if err != nil {
		log.WithError(err).Debugln("Decode block hash failed.")
		return nil, err
	}

	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()

	block, err := chain.GetBlockByHash((*common.Hash)(byteHash))

	if err != nil {
		log.WithError(err).WithField("hash",
			hash).Debugln("Get block by hash failed.")
	}

	respBlock := utils.KarmemBlock2Protobuf(block, *full)
	resp = new(pb.GetBlockResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Body = respBlock

	return resp, nil
}

func (s *blockchainService) GetBlockByNumber(ctx context.Context,
	in *pb.GetBlockReq) (resp *pb.GetBlockResp, err error) {
	// todo: validate the hash value
	height := in.Number
	full := in.Full

	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()

	block, err := chain.GetBlockByHeight(int64(*height))

	if err != nil {
		log.WithError(err).WithField("height",
			height).Debugln("Get block by height failed.")
	}

	respBlock := utils.KarmemBlock2Protobuf(block, *full)
	resp = new(pb.GetBlockResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Body = respBlock

	return resp, nil
}

func (s *blockchainService) GetTransactionByHash(ctx context.Context,
	in *pb.GetTransactionReq) (resp *pb.GetTransactionResp, err error) {
	panic("Unimplemented")
}

func (s *blockchainService) GetTransactionByBlockHashAndIndex(ctx context.
	Context, in *pb.GetTransactionReq) (resp *pb.GetTransactionResp,
	err error) {
	panic("Unimplemented")
}

func (s *blockchainService) GetTransactionByBlockNumberAndIndex(ctx context.
	Context, in *pb.GetTransactionReq) (resp *pb.GetTransactionResp,
	err error) {
	panic("Unimplemented")
}
