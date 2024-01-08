/**
  @author: decision
  @date: 2023/11/29
  @note:
**/

package utils

import (
	"encoding/hex"
	"github.com/chain-lab/go-chronos/common"
	"github.com/gogo/protobuf/proto"
	"time"
)
import "github.com/chain-lab/go-chronos/rpc/pb"

func KarmemBlock2Protobuf(block *common.Block, full bool) *pb.Block {
	pbBlock := new(pb.Block)
	pbBlockHeader := new(pb.BlockHeader)

	pbBlockHeader.Timestamp = proto.Uint64(uint64(time.Now().Unix()))
	pbBlockHeader.PrevBlockHash = proto.String("0x" + block.PrevBlockHash())
	pbBlockHeader.BlockHash = proto.String("0x" + block.BlockHash())
	pbBlockHeader.MerkleRoot = proto.String("0x" + hex.EncodeToString(block.Header.
		MerkleRoot[:]))
	pbBlockHeader.Height = proto.Uint64(uint64(block.Header.Height))
	pbBlockHeader.Public = proto.String("0x" + hex.EncodeToString(block.Header.
		PublicKey[:]))
	pbBlockHeader.Params = proto.String("0x" + hex.EncodeToString(block.Header.Params))
	pbBlockHeader.GasLimit = proto.Uint64(0)

	pbBlock.Header = pbBlockHeader

	if !full {
		return pbBlock
	}

	pbBlock.Transactions = make([]*pb.Transaction, 0)
	for _, tx := range block.Transactions {
		pbBlock.Transactions = append(pbBlock.Transactions, KarmemTransaction2Protobuf(&tx))
	}

	return pbBlock
}

func KarmemTransaction2Protobuf(tx *common.Transaction) *pb.Transaction {
	pbTransaction := new(pb.Transaction)

	pbTransaction.Hash = proto.String("0x" + hex.EncodeToString(tx.Body.Hash[:]))
	pbTransaction.Address = proto.String("0x" + hex.EncodeToString(tx.Body.Address[:]))
	pbTransaction.Receiver = proto.String("0x" + hex.EncodeToString(tx.Body.Receiver[:]))
	pbTransaction.Gas = proto.Uint64(uint64(tx.Body.Gas))
	pbTransaction.Nonce = proto.Uint64(uint64(tx.Body.Nonce))
	pbTransaction.Event = proto.String("0x" + hex.EncodeToString(tx.Body.Event))
	pbTransaction.Opt = proto.String("0x" + hex.EncodeToString(tx.Body.Opt))
	pbTransaction.State = proto.String("0x" + hex.EncodeToString(tx.Body.State))
	pbTransaction.Data = proto.String("0x" + hex.EncodeToString(tx.Body.Data))
	pbTransaction.Expire = proto.Uint64(uint64(tx.Body.Expire))
	pbTransaction.Timestamp = proto.Uint64(uint64(tx.Body.Timestamp))
	pbTransaction.Public = proto.String("0x" + hex.EncodeToString(tx.Body.Public[:]))
	pbTransaction.Signature = proto.String("0x" + hex.EncodeToString(tx.Body.Signature))
	pbTransaction.Height = proto.Uint64(uint64(tx.Body.Height))
	pbTransaction.BlockHash = proto.String("0x" + hex.EncodeToString(tx.Body.
		BlockHash[:]))
	pbTransaction.Index = proto.Uint64(uint64(tx.Body.Index))

	return pbTransaction
}
