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
	//LoadConfig("./config.yml")
	addresses := config.Strings("rpc.address")

	for {
		// 随机选取列表中的节点
		idx := rand2.Intn(len(addresses))
		addr := addresses[idx]
		log.Infof("Select host %s", addr)

		// 连接到选取的节点
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.WithField("error", err).Errorln("Start connect failed.")
			continue
		}

		log.Infoln("Connect to host %s", addr)

		// 利用 conn 创建 RPC 客户端
		c := pb.NewTransactionServiceClient(conn)

		// 设置超时时间为3秒，超过时间后断开，再选取新的节点连接
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		// 随机生成私钥
		prv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			log.WithError(err).Errorln("Generate new key failed.")
			cancel()
			continue
		}

		count := 0
		log.Infof("Start send transactions.")
		for {
			// 构建新的交易
			tx := buildTransaction(prv)
			bytesTransaction, err := utils.SerializeTransaction(tx)
			if err != nil {
				log.WithError(err).Errorln("Build transaction failed.")
				break
			}

			// 将字节类型的交易编码
			encodedTransaction := hex.EncodeToString(bytesTransaction)
			// 通过 client 向服务端提交交易
			resp, err := c.SubmitTransaction(ctx, &pb.SubmitTransactionReq{
				SignedTransaction: proto.String(encodedTransaction),
			})
			if err != nil {
				log.WithError(err).Errorln(
					"Signed transaction send failed.")
				break
			}

			if count >= 3000 || resp.GetStatus() == pb.
				SubmitTransactionStatus_Default {
				// 断线重连，如果返回状态为 default 说明本次 rpc 连接断开（原因？）
				conn.Close()
				log.Errorln("Receive code default or upper to limit.")
				break
			}
			count += 1
		}
		cancel()
	}
}

func LoadConfig(filepath string) {
	config.WithOptions(config.ParseEnv)

	config.AddDriver(yaml.Driver)

	err := config.LoadFiles(filepath)
	if err != nil {
		log.WithField("error", err).Errorln("Load config file failed.")
	}
}
