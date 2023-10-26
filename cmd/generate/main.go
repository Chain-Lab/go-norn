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
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/libp2p/go-libp2p/core/crypto"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

func main() {
	config.WithOptions(
		config.ParseEnv,
		config.SaveFileOnSet("./config.yml", "yml"),
	)

	config.AddDriver(yaml.Driver)

	err := config.LoadFiles("./config.yml")
	if err != nil {
		log.WithField("error", err).Errorln("Load config file failed.")
	}

	prv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	config.Set("consensus.prv", hex.EncodeToString(prv.D.Bytes()))

	sha3 := sha3.New256()
	sha3.Write(elliptic.MarshalCompressed(elliptic.P256(), prv.X, prv.Y))
	config.Set("consensus.pub", hex.EncodeToString(elliptic.MarshalCompressed(elliptic.P256(), prv.X, prv.Y)))
	config.Set("consensus.address", hex.EncodeToString(sha3.Sum(nil))[0:40])

	priv, _, _ := crypto.GenerateECDSAKeyPairWithCurve(elliptic.P256(), rand.Reader)
	data, _ := crypto.MarshalPrivateKey(priv)
	config.Set("p2p.prv", hex.EncodeToString(data))
	//config.Set("p2p.bootstrap", )

	config.Set("node.port", 31258)
	config.Set("node.udp", 31259)
	config.Set("rpc.address", "0.0.0.0:45555")
	config.Set("metrics.port", 8700)
}
