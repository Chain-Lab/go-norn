package utils

import (
	"github.com/chain-lab/go-chronos/common"
	"strconv"
)

func BlockHash2DBKey(hash common.Hash) []byte {
	return append([]byte("block#"), hash[:]...)
}

func BlockHeight2DBKey(height int64) []byte {
	strHeight := strconv.FormatInt(height, 10)

	return append([]byte("block#"), []byte(strHeight)...)
}

func TxHash2DBKey(hash common.Hash) []byte {
	return append([]byte("tx#"), hash[:]...)
}
