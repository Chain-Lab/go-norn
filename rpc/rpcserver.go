/**
  @author: decision
  @date: 2023/6/16
  @note:
**/

package rpc

import (
	"github.com/chain-lab/go-chronos/rpc/pb"
	"github.com/gookit/config/v2"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

func RPCServerStart() {
	// 从配置文件中获取 RPC 绑定的 ip 和 port
	addr := config.String("rpc.address")
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.WithField("error", err).Errorln("RPC listen port failed.")
		return
	}

	limiter := NewRateLimiter(3000)

	s := grpc.NewServer(
		grpc.UnaryInterceptor(limiter.UnaryInterceptor),
	)
	// 注册 RPC 处理的 Service
	pb.RegisterTransactionServiceServer(s, &transactionService{})
	pb.RegisterNodeServer(s, &nodeService{})
	pb.RegisterBlockchainServer(s, &blockchainService{})
	reflection.Register(s)

	log.Traceln("RPC server started.")

	err = s.Serve(lis)
	if err != nil {
		log.WithField("error", err).Errorln("RPC server failed.")
		return
	}
}
