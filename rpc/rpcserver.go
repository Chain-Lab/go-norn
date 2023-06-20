/**
  @author: decision
  @date: 2023/6/16
  @note:
**/

package rpc

import (
	"github.com/gookit/config/v2"
	log "github.com/sirupsen/logrus"
	"go-chronos/rpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
)

func RPCServerStart() {
	addr := config.String("rpc.address")
	lis, err := net.Listen("tcp", addr)

	if err != nil {
		log.WithField("error", err).Errorln("RPC listen port failed.")
		return
	}

	s := grpc.NewServer()
	pb.RegisterTransactionServer(s, &transactionService{})
	reflection.Register(s)

	log.Infoln("RPC server started.")

	err = s.Serve(lis)
	if err != nil {
		log.WithField("error", err).Errorln("RPC server failed.")
		return
	}
}
