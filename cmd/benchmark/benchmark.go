package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/core"
	"time"
)

func BuildAndVerifyTransaction() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.WithField("error", err).Panicln("Generate private key failed.")
		return
	}

	buildStart := time.Now()
	transaction := buildTransaction(privateKey)
	buildTimeUsed := time.Since(buildStart)

	verifyStart := time.Now()
	transaction.Verify()
	verifyTimeUsed := time.Since(verifyStart)

	log.Infof("Build transaction use %d us.", buildTimeUsed.Microseconds())
	log.Infof("Verify transaction use %d us.", verifyTimeUsed.Microseconds())
}

func BuildAndVerifyMassiveTransaction() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		log.WithField("error", err).Panicln("Generate private key failed.")
		return
	}

	txs := make([]common.Transaction, 3000)

	buildStart := time.Now()
	for i := 0; i < 3000; i++ {
		tx := buildTransaction(privateKey)
		txs = append(txs, *tx)
	}
	buildTimeUsed := time.Since(buildStart)

	verifyStart := time.Now()
	for i := 0; i < 3000; i++ {
		tx := txs[i]
		tx.Verify()
	}
	verifyTimeUsed := time.Since(verifyStart)

	log.Infof("Build 3000 transactions use %d us.", buildTimeUsed)
	log.Infof("Verify 3000 transactions use %d us.", verifyTimeUsed)

	log.Infof("Build transaction average use %d us.", buildTimeUsed/3000)
	log.Infof("Verify transaction average use %d us.", verifyTimeUsed/3000)

}

func PackageBlockAndInsert() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		log.WithField("error", err).Panicln("Generate private key failed.")
		return
	}

	txs := make([]common.Transaction, 3000)

	for i := 0; i < 3000; i++ {
		tx := buildTransaction(privateKey)
		txs = append(txs, *tx)
	}
	for i := 0; i < 3000; i++ {
		tx := txs[i]
		tx.Verify()
	}

	bc := core.GetBlockChainInst()
	packageStart := time.Now()
	block, err := bc.PackageNewBlock(txs)
	packageTimeUsed := time.Since(packageStart)

	if err != nil {
		log.WithField("error", err).Panicln("Package block failed.")
		return
	}

	insertStart := time.Now()
	err = bc.InsertBlock(block)
	if err != nil {
		log.WithField("error", err).Panicln("Insert block failed.")
		return
	}
	insertTimeUse := time.Since(insertStart)

	log.Infof("Package block use %d us.", packageTimeUsed.Microseconds())
	log.Infof("Insert block use %d us.", insertTimeUse.Microseconds())

}

func main() {
	BuildAndVerifyTransaction()
	BuildAndVerifyMassiveTransaction()
	PackageBlockAndInsert()
}
