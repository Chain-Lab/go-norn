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
	pbBlockHeader.PrevBlockHash = proto.String(block.PrevBlockHash())
	pbBlockHeader.BlockHash = proto.String(block.BlockHash())
	pbBlockHeader.MerkleRoot = proto.String(hex.EncodeToString(block.Header.
		MerkleRoot[:]))
	pbBlockHeader.Height = proto.Uint64(uint64(block.Header.Height))
	pbBlockHeader.Public = proto.String(hex.EncodeToString(block.Header.
		PublicKey[:]))
	pbBlockHeader.Params = proto.String(hex.EncodeToString(block.Header.Params))
	pbBlockHeader.GasLimit = proto.Uint64(0)

	pbBlock.Header = pbBlockHeader

	if !full {
		return pbBlock
	}

	return pbBlock
}
