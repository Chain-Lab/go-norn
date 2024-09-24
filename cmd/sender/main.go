/**
  @author: decision
  @date: 2023/6/20
  @note: 交易发送模拟代码
**/

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/chain-lab/go-norn/rpc/pb"
	"github.com/chain-lab/go-norn/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	log "github.com/sirupsen/logrus"
	rand2 "golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func main() {
	LoadConfig("./config.yml")
	addresses := config.Strings("rpc.address")

	// 随机选取列表中的节点
	idx := rand2.Intn(len(addresses))
	addr := addresses[idx]
	log.Infof("Select host %s", addr)

	// 连接到选取的节点
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithField("error", err).Errorln("Start connect failed.")
		return
	}
	log.Infoln("Connect to host %s", addr)

	// 创建 RPC 客户端
	c := pb.NewTransactionServiceClient(conn)

	// 设置超时时间为3秒
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 随机生成私钥
	prv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.WithError(err).Errorln("Generate new key failed.")
		return
	}

	// 构建并发送单个交易
	tx := buildTransaction(prv)
	bytesTransaction, err := utils.SerializeTransaction(tx)
	if err != nil {
		log.WithError(err).Errorln("Build transaction failed.")
		return
	}

	fmt.Println(tx.Body.Hash[:])

	// 编码并提交交易
	encodedTransaction := hex.EncodeToString(bytesTransaction)
	resp, err := c.SubmitTransaction(ctx, &pb.SubmitTransactionReq{
		SignedTransaction: proto.String(encodedTransaction),
	})
	if err != nil {
		log.WithError(err).Errorln("Signed transaction send failed.")
		return
	}

	log.Infof("Transaction submitted, status: %s", resp.GetStatus())
	cancel()
}

func LoadConfig(filepath string) {
	config.WithOptions(config.ParseEnv)

	config.AddDriver(yaml.Driver)

	err := config.LoadFiles(filepath)
	if err != nil {
		log.WithField("error", err).Errorln("Load config file failed.")
	}
}
