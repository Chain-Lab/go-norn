/**
  @author: decision
  @date: 2023/5/18
  @note:
**/

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/sha3"
)

func main() {
	prv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	println(prv.D.BitLen())
	println(hex.EncodeToString(prv.D.Bytes()))

	sha3 := sha3.New256()
	sha3.Write(elliptic.MarshalCompressed(elliptic.P256(), prv.X, prv.Y))
	println(hex.EncodeToString(elliptic.MarshalCompressed(elliptic.P256(), prv.X, prv.Y)))
	println(hex.EncodeToString(sha3.Sum(nil))[0:40])

	priv, _, _ := crypto.GenerateECDSAKeyPairWithCurve(elliptic.P256(), rand.Reader)
	data, _ := crypto.MarshalPrivateKey(priv)
	println(len(data))
	println(hex.EncodeToString(data))

}
