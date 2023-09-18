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
	"github.com/chain-lab/go-chronos/rpc/pb"
	"github.com/chain-lab/go-chronos/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

func main() {
	LoadConfig("./config.yml")
	addr := config.String("rpc.address")

	for {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			log.WithField("error", err).Errorln("Start connect failed.")
			return
		}

		c := pb.NewTransactionClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		prv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

		ticker := time.NewTicker(time.Microsecond * 100)
		reconnect := false
		for {
			select {
			case <-ticker.C:
				tx := buildTransaction(prv)
				bytesTransaction, err := utils.SerializeTransaction(tx)

				if err != nil {
					continue
				}

				encodedTransaction := hex.EncodeToString(bytesTransaction)
				resp, err := c.SubmitTransaction(ctx, &pb.SubmitTransactionReq{
					//_, err = c.SubmitTransaction(ctx, &pb.SubmitTransactionReq{
					SignedTransaction: proto.String(encodedTransaction),
				})

				if resp.GetStatus() == pb.SubmitTransactionStatus_Default {
					// 断线重连，如果返回状态为 default 说明本次 rpc 连接断开（原因？）
					reconnect = true
				}
			}

			if reconnect {
				conn.Close()
				cancel()
				break
			}
			//log.Infoln(resp.GetStatus())
		}
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
