package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/crypto"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"time"
)

func buildTransaction(key *ecdsa.PrivateKey) *common.Transaction {
	// 生成随机数据
	data := make([]byte, 32)
	rand.Read(data)
	timestamp := time.Now().UnixMilli()

	// 构建交易体
	txBody := common.TransactionBody{
		Data:      data,
		Timestamp: timestamp,
		Expire:    timestamp + 3000,
	}

	// 设置交易的公钥、地址，初始化哈希值、签名为空
	txBody.Public = [33]byte(crypto.PublicKey2Bytes(&key.PublicKey))
	txBody.Address = crypto.PublicKeyBytes2Address(txBody.Public)
	txBody.Hash = [32]byte{}
	txBody.Signature = []byte{}

	// 将当前未签名的交易进行序列化 -> 字节形式
	writer := karmem.NewWriter(1024)
	txBody.WriteAsRoot(writer)
	txBodyBytes := writer.Bytes()

	// 哈希序列化后的交易，然后签名
	hash := sha256.New()
	hash.Write(txBodyBytes)
	txHashBytes := hash.Sum(nil)
	txSignatureBytes, err := ecdsa.SignASN1(rand.Reader, key, txHashBytes)

	if err != nil {
		log.WithField("error", err).Errorln("Sign transaction failed.")
		return nil
	}

	// 写入签名和哈希信息
	txBody.Hash = [32]byte(txHashBytes)
	txBody.Signature = txSignatureBytes
	tx := common.Transaction{
		Body: txBody,
	}

	return &tx
}
