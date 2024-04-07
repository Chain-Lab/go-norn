/**
  @author: decision
  @date: 2023/11/28
  @note:
**/

package rpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/crypto"
	"github.com/chain-lab/go-norn/node"
	"github.com/chain-lab/go-norn/rpc/pb"
	"github.com/chain-lab/go-norn/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/gookit/config/v2"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	karmem "karmem.org/golang"
	"time"
)

const (
	setCommandString    = "set"
	appendCommandString = "append"
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

	if hash == nil {
		return nil, fmt.Errorf("block hash is required")
	}

	strHash := removePrefixIfExists(*hash)
	byteHash, err := hex.DecodeString(strHash)

	if err != nil {
		log.WithError(err).Debugln("Decode block hash failed.")
		return nil, err
	}

	pm := node.GetP2PManager()
	// todo: check pm and chain is null
	chain := pm.GetBlockChain()

	block, err := chain.GetBlockByHash((*common.Hash)(byteHash))

	if err != nil || block == nil {
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

	strHash := removePrefixIfExists(*txHash)
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

	resp.Hex = proto.String(string(data))
	return resp, nil
}

func (s *blockchainService) SendTransactionWithData(ctx context.
	Context, in *pb.SendTransactionWithDataReq) (resp *pb.SendTransactionWithDataResp,
	err error) {
	receiver := in.Receiver
	if receiver == nil {
		return nil, fmt.Errorf("transaction receiver is required")
	}

	if in.Type == nil || (*in.Type != setCommandString && *in.
		Type != appendCommandString) {
		return nil, fmt.Errorf("transaction type error")
	}

	// todo: build transaction as a function
	prvHex := config.String("consensus.prv")
	prv, err := crypto.DecodePrivateKeyFromHexString(prvHex)

	if err != nil {
		log.WithError(err).Debugln("Decode private key from hex string failed.")
		return nil, err
	}

	timestamp := time.Now().UnixMilli()

	address, err := hex.DecodeString(*receiver)
	if err != nil {
		log.WithError(err).Errorln("Decode address failed.")
		return
	}

	// 构建交易体
	txBody := common.TransactionBody{
		Data:      nil,
		Receiver:  [20]byte(address),
		Timestamp: timestamp,
		Expire:    timestamp + 3000,
	}

	txBody.Public = [33]byte(crypto.PublicKey2Bytes(&prv.PublicKey))
	txBody.Address = crypto.PublicKeyBytes2Address(txBody.Public)
	txBody.Hash = [32]byte{}
	txBody.Signature = []byte{}

	if in.Key != nil && in.Value != nil {
		dataCmd := common.DataCommand{
			Opt:   []byte(*in.Type),
			Key:   []byte(*in.Key),
			Value: []byte(*in.Value),
		}

		writer := karmem.NewWriter(1024)
		dataCmd.WriteAsRoot(writer)
		txBody.Data = writer.Bytes()
	}

	// 将当前未签名的交易进行序列化 -> 字节形式
	writer := karmem.NewWriter(1024)
	txBody.WriteAsRoot(writer)
	txBodyBytes := writer.Bytes()

	hash := sha256.New()
	hash.Write(txBodyBytes)
	txHashBytes := hash.Sum(nil)
	txSignatureBytes, err := ecdsa.SignASN1(rand.Reader, prv, txHashBytes)

	if err != nil {
		log.WithField("error", err).Errorln("Sign transaction failed.")
		return
	}

	// 写入签名和哈希信息
	txBody.Hash = [32]byte(txHashBytes)
	txBody.Signature = txSignatureBytes
	tx := &common.Transaction{
		Body: txBody,
	}

	pm := node.GetP2PManager()
	pm.AddTransaction(tx)

	resp = new(pb.SendTransactionWithDataResp)
	resp.TxHash = proto.String(hex.EncodeToString(tx.Body.Hash[:]))

	return resp, nil
}

func removePrefixIfExists(hexString string) string {
	if hexString[:2] == "0x" {
		return hexString[2:]
	} else {
		return hexString
	}
}
