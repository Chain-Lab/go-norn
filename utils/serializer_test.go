package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/crypto"
	karmem "karmem.org/golang"
	"testing"
	"time"
)

func buildTransaction(key *ecdsa.PrivateKey) *common.Transaction {
	data := make([]byte, 32)
	rand.Read(data)
	timestamp := time.Now().UnixMilli()

	txBody := common.TransactionBody{
		Data:      data,
		Timestamp: timestamp,
		Expire:    timestamp + 3000,
	}

	txBody.Public = [33]byte(crypto.PublicKey2Bytes(&key.PublicKey))
	txBody.Address = crypto.PublicKeyBytes2Address(txBody.Public)
	txBody.Hash = [32]byte{}
	txBody.Signature = []byte{}

	writer := karmem.NewWriter(1024)
	txBody.WriteAsRoot(writer)
	txBodyBytes := writer.Bytes()

	hash := sha256.New()
	hash.Write(txBodyBytes)
	txHashBytes := hash.Sum(nil)
	txSignatureBytes, err := ecdsa.SignASN1(rand.Reader, key, txHashBytes)

	if err != nil {
		log.WithField("error", err).Errorln("Sign transaction failed.")
		return nil
	}

	txBody.Hash = [32]byte(txHashBytes)
	txBody.Signature = txSignatureBytes
	tx := common.Transaction{
		Body: txBody,
	}

	return &tx
}

func TestSerializeTransaction(t *testing.T) {
	prv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		t.Fatal("Generate keypair failed.")
	}

	tx := buildTransaction(prv)

	txByteData, err := SerializeTransaction(tx)
	if err != nil {
		t.Fatal(err)
	}

	txCopyed, err := DeserializeTransaction(txByteData)
	if err != nil {
		t.Fatal(err)
	}

	if !txCopyed.Verify() {
		t.Fatal("Verify transaction failed.")
	}
}
