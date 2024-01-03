/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package utils

import "encoding/hex"

func ByteAddressToHexString(address []byte) string {
	return hex.EncodeToString(address)
}
