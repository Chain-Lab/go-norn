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

// DecodePrivateKeyFromHexString
//
//	@Description: 从 16 进制的字符串转换为私钥类型，同时计算它的公钥
//	@param hexString - 16 进制字符串
//	@return *ecdsa.PrivateKey - 私钥
//	@return error - 报错信息
func DecodePrivateKeyFromHexString(hexString string) (*ecdsa.PrivateKey,
	error) {
	result := new(ecdsa.PrivateKey)
	k := new(big.Int)

	kBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}

	k.SetBytes(kBytes)
	result.D = k

	x, y := elliptic.P256().ScalarBaseMult(kBytes)

	result.PublicKey.X = x
	result.PublicKey.Y = y
	result.Curve = elliptic.P256()
	return result, nil
}
