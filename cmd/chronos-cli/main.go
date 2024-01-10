/**
  @author: decision
  @date: 2023/12/22
  @note:
**/

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/crypto"
	"github.com/chain-lab/go-chronos/rpc/pb"
	"github.com/chain-lab/go-chronos/utils"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	karmem "karmem.org/golang"
	"os"
	"time"
)

const (
	addr = "localhost:45555"
	//addr = "43.156.111.199:45555"
)

func main() {
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendAddress := sendCmd.String("receiver", "", "receiver address")
	sendKey := sendCmd.String("key", "", "transaction set key")
	sendValue := sendCmd.String("value", "", "transaction set value")

	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getAddress := getCmd.String("address", "", "data storage address")
	getKey := getCmd.String("key", "", "data storage key")

	if len(os.Args) < 2 {
		fmt.Printf("expected 'send' or 'get' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "send":
		sendCmd.Parse(os.Args[2:])

		if *sendAddress == "" {
			log.Errorln("Argument receiver address is required.")
			return
		}

		executeSendCommand(*sendAddress, *sendKey, *sendValue)
	case "get":
		getCmd.Parse(os.Args[2:])
		executeGetCommand(*getAddress, *getKey)
	}
}

func executeSendCommand(receiver, key, value string) {
	prv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.WithError(err).Errorln("Generate random key failed.")
		return
	}

	timestamp := time.Now().UnixMilli()

	address, err := hex.DecodeString(receiver)
	if err != nil {
		log.WithError(err).Errorln("Decode address failed.")
		return
	}

	// 构建交易体
	txBody := common.TransactionBody{
		Data:      nil,
		Receiver:  [20]byte(address),
		Timestamp: timestamp,
		Expire:    timestamp + 3000,
	}

	txBody.Public = [33]byte(crypto.PublicKey2Bytes(&prv.PublicKey))
	txBody.Address = crypto.PublicKeyBytes2Address(txBody.Public)
	txBody.Hash = [32]byte{}
	txBody.Signature = []byte{}

	if key != "" {
		dataCmd := common.DataCommand{
			Opt:   []byte("set"),
			Key:   []byte(key),
			Value: []byte(value),
		}

		writer := karmem.NewWriter(1024)
		dataCmd.WriteAsRoot(writer)
		txBody.Data = writer.Bytes()
	}

	// 将当前未签名的交易进行序列化 -> 字节形式
	writer := karmem.NewWriter(1024)
	txBody.WriteAsRoot(writer)
	txBodyBytes := writer.Bytes()

	hash := sha256.New()
	hash.Write(txBodyBytes)
	txHashBytes := hash.Sum(nil)
	txSignatureBytes, err := ecdsa.SignASN1(rand.Reader, prv, txHashBytes)

	if err != nil {
		log.WithField("error", err).Errorln("Sign transaction failed.")
		return
	}

	// 写入签名和哈希信息
	txBody.Hash = [32]byte(txHashBytes)
	txBody.Signature = txSignatureBytes
	tx := &common.Transaction{
		Body: txBody,
	}

	bytesTransaction, err := utils.SerializeTransaction(tx)
	encodedTransaction := hex.EncodeToString(bytesTransaction)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithField("error", err).Errorln("Start connect failed.")
		return
	}

	c := pb.NewTransactionServiceClient(conn)

	// 设置超时时间为3秒，超过时间后断开，再选取新的节点连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	_, err = c.SubmitTransaction(ctx, &pb.SubmitTransactionReq{
		SignedTransaction: proto.String(encodedTransaction),
	})

	if err != nil {
		log.WithError(err).Errorln(
			"Signed transaction send failed.")
		cancel()
		return
	}
	log.Infof("Transaction 0x%s submitted.", hex.EncodeToString(tx.Body.
		Hash[:]))

	cancel()
}

func executeGetCommand(address, key string) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithField("error", err).Errorln("Start connect failed.")
		return
	}

	c := pb.NewBlockchainClient(conn)

	// 设置超时时间为3秒，超过时间后断开，再选取新的节点连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	resp, err := c.ReadContractAddress(ctx, &pb.ReadContractAddressReq{
		Address: proto.String(address),
		Key:     proto.String(key),
	})

	if err != nil {
		log.WithError(err).Errorln("Read address storage data failed.")
		cancel()
		return
	}

	decoded, err := hex.DecodeString(*resp.Hex)
	if err != nil {
		log.WithError(err).Errorln("Decode result failed.")
		return
	}
	log.Infof("Read data result: %s", string(decoded))

	cancel()

}
