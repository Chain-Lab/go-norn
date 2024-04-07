package utils

import (
	"encoding/hex"
	"fmt"
	"github.com/chain-lab/go-norn/common"
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

func DataAddressKey2DBKey(address, key []byte) []byte {
	dbKey := fmt.Sprintf("data#%s#%s", hex.EncodeToString(address), string(key))
	return []byte(dbKey)
}
