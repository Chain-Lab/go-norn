package common

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"sync"
)

var writerPool = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}

func (t *Transaction) Verify() bool {
	writer := writerPool.Get().(*karmem.Writer)
	defer writer.Reset()
	defer writerPool.Put(writer)

	txBody := t.Body
	byteSignature := make([]byte, len(txBody.Signature))
	bytePublicKey := txBody.Public[:]
	// todo: 这样将 [32]byte 转换为 []byte 是否存在风险
	byteHash := make([]byte, 32)
	copy(byteSignature, txBody.Signature[:])
	copy(byteHash, txBody.Hash[:])

	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), bytePublicKey[:])
	publicKey := ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	txBody.Signature = []byte{}
	txBody.Hash = [32]byte{}

	if _, err := txBody.WriteAsRoot(writer); err != nil {
		log.WithField("error", err).Debugln("Karmem writer error.")
		return false
	}

	hash := sha256.New()
	hash.Write(writer.Bytes())
	txHashBytes := hash.Sum(nil)

	if bytes.Compare(txHashBytes, byteHash) != 0 {
		log.Debugln("Transaction hash not match.")
		return false
	}

	return ecdsa.VerifyASN1(&publicKey, txHashBytes, byteSignature)
}
