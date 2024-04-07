// Package crypto
// @Description: 和密钥转换相关的函数
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha1"
)

// PublicKey2Bytes
//
//	@Description: 将椭圆曲线上的点 (x,y) 编码为 33 bytes 的公钥
//	@param pub - ecdsa 类型的公钥
//	@return []byte - 编码后的公钥
func PublicKey2Bytes(pub *ecdsa.PublicKey) []byte {
	x, y := pub.X, pub.Y
	marshalBytes := elliptic.MarshalCompressed(elliptic.P256(), x, y)
	return marshalBytes
}

// PublicKeyBytes2Address
//
//	@Description: 公钥 -> 地址
//	@param pubBytes
//	@return [20]byte
func PublicKeyBytes2Address(pubBytes [33]byte) [20]byte {
	data := pubBytes[:]
	hash := sha1.New()
	hash.Write(data)
	sliceBytes := hash.Sum(nil)
	return [20]byte(sliceBytes)
}

// Bytes2PublicKey
//
//	@Description: 字节码类型的地址 -> 公钥实例
//	@param pub
//	@return *ecdsa.PublicKey
func Bytes2PublicKey(pub []byte) *ecdsa.PublicKey {
	// 这里统一使用的是 p256 曲线
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), pub)
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
}
