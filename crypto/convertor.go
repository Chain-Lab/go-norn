package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha1"
)

func PublicKey2Bytes(pub *ecdsa.PublicKey) []byte {
	x, y := pub.X, pub.Y
	marshalBytes := elliptic.MarshalCompressed(elliptic.P256(), x, y)
	return marshalBytes
}

func PublicKeyBytes2Address(pubBytes [33]byte) [20]byte {
	data := pubBytes[:]
	hash := sha1.New()
	hash.Write(data)
	sliceBytes := hash.Sum(nil)
	return [20]byte(sliceBytes)
}

func Bytes2PublicKey(pub []byte) *ecdsa.PublicKey {
	// 这里统一使用的是 p256 曲线
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), pub)
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
}
