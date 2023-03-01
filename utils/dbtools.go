package utils

import (
	"go-chronos/common"
	"strconv"
)

func BlockHash2DBKey(hash common.Hash) []byte {
	return append([]byte("block#"), hash[:]...)
}

func BlockHeight2DBKey(height int) []byte {
	strHeight := strconv.Itoa(height)

	return append([]byte("block#"), []byte(strHeight)...)
}
