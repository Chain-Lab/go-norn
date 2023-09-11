/**
  @author: decision
  @date: 2023/9/11
  @note:
**/

package benchmark

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	log "github.com/sirupsen/logrus"
	"testing"
)

func BenchmarkPackageBlock(b *testing.B) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.WithField("error", err).Panicln("Generate private key failed.")
		return
	}

	transaction := buildTransaction(privateKey)
	transaction.Verify()
}
