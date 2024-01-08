/**
  @author: decision
  @date: 2023/11/28
  @note:
**/

package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/node"
	"github.com/chain-lab/go-chronos/rpc/pb"
	"github.com/chain-lab/go-chronos/utils"
	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"strings"
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
	height := in.Number
	full := in.Full

	if height == nil {
		return nil, fmt.Errorf("block height is required")
	}

	// 在 full 变量为空的时候设置为 false
	if full == nil {
		full = proto.Bool(false)
	}
	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()

	block, err := chain.GetBlockByHeight(int64(*height))

	if err != nil {
		log.WithError(err).WithField("height",
			height).Debugln("Get block by height failed.")
		return nil, err
	}

	respBlock := utils.KarmemBlock2Protobuf(block, *full)
	resp = new(pb.GetBlockResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Body = respBlock

	return resp, nil
}

func (s *blockchainService) GetTransactionByHash(ctx context.Context,
	in *pb.GetTransactionReq) (resp *pb.GetTransactionResp, err error) {
	txHash := in.Hash
	if txHash == nil {
		return nil, fmt.Errorf("transaction hash is empty")
	}

	strHash := strings.Trim(*txHash, "0x")
	pm := node.GetP2PManager()
	chain := pm.GetBlockChain()
	hash, err := hex.DecodeString(strHash)
	if err != nil {
		log.WithError(err).Debugln("Transaction hash to common hash failed.")
		return nil, err
	}

	tx, err := chain.GetTransactionByHash(common.Hash(hash))
	if err != nil {
		log.WithError(err).Debugln("Get transaction by hash failed.")
		return nil, err
	}

	resp = new(pb.GetTransactionResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Body = utils.KarmemTransaction2Protobuf(tx)

	return resp, nil
}

func (s *blockchainService) GetTransactionByBlockHashAndIndex(ctx context.
Context, in *pb.GetTransactionReq) (resp *pb.GetTransactionResp,
	err error) {
	// todo: validate the hash value
	hash := in.BlockHash
	index := in.Index

	if hash == nil || index == nil {
		return nil, fmt.Errorf("block hash or index required.")
	}

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

	if *index >= uint64(len(block.Transactions)) {
		return nil, fmt.Errorf("index out of range")
	}

	tx := block.Transactions[*index]
	resp = new(pb.GetTransactionResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Body = utils.KarmemTransaction2Protobuf(&tx)

	return resp, nil
}

func (s *blockchainService) GetTransactionByBlockNumberAndIndex(ctx context.
Context, in *pb.GetTransactionReq) (resp *pb.GetTransactionResp,
	err error) {
	height := in.BlockNumber
	index := in.Index

	if height == nil || index == nil {
		return nil, fmt.Errorf("block hash or index required.")
	}

	if height == nil {
		return nil, fmt.Errorf("block height is required")
	}

	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()

	block, err := chain.GetBlockByHeight(int64(*height))

	if err != nil {
		log.WithError(err).WithField("height",
			height).Debugln("Get block by height failed.")
		return nil, err
	}

	if *index >= uint64(len(block.Transactions)) {
		return nil, fmt.Errorf("index out of range")
	}

	tx := block.Transactions[*index]
	resp = new(pb.GetTransactionResp)
	resp.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	resp.Body = utils.KarmemTransaction2Protobuf(&tx)

	return resp, nil
}

func (s *blockchainService) ReadContractAddress(ctx context.
Context, in *pb.ReadContractAddressReq) (resp *pb.ReadContractAddressResp,
	err error) {
	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()
	address := in.Address
	key := in.Key

	resp = new(pb.ReadContractAddressResp)

	data, err := chain.ReadAddressData(*address, *key)

	if err != nil {
		log.WithError(err).Debugln("Read blockchain data failed.")
		return nil, err
	}

	resp.Hex = proto.String(hex.EncodeToString(data))
	return resp, nil
}
