package common

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"github.com/chain-lab/go-chronos/crypto"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"testing"
	"time"
)

func buildTransaction(key *ecdsa.PrivateKey) *Transaction {
	data := make([]byte, 32)
	rand.Read(data)
	timestamp := time.Now().UnixMilli()

	txBody := TransactionBody{
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
	tx := Transaction{
		Body: txBody,
	}

	return &tx
}

func TestBuildTransactionAndVerifyTransaction(t *testing.T) {
	//log.SetLevel(log.DebugLevel)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		t.Fatal(err)
	}
	st := time.Now()
	transaction := buildTransaction(privateKey)
	buildTimeUsed := time.Since(st)

	st = time.Now()
	result := transaction.Verify()
	verifyTimeUsed := time.Since(st)
	if !result {
		t.Fatal("Verify transaction failed.")
	}

	t.Logf("Build transaction use %d μs", buildTimeUsed.Microseconds())
	t.Logf("Verify transaction use %d μs", verifyTimeUsed.Microseconds())
}
