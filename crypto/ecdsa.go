/**
  @author: decision
  @date: 2023/5/24
  @note:
**/

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"math/big"
)

func decodePrivateKeyFromHexString(hexString string) (*ecdsa.PrivateKey, error) {
	result := new(ecdsa.PrivateKey)
	k := new(big.Int)

	kBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}

	k.SetBytes(kBytes)
	result.D = k

	x, y := elliptic.P256().ScalarBaseMult(kBytes)

	result.X = x
	result.Y = y
	return result, nil
}
