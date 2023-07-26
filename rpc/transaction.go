/**
  @author: decision
  @date: 2023/6/16
  @note:
**/

package rpc

import (
	"context"
	"encoding/hex"
	"github.com/gogo/protobuf/proto"
	"go-chronos/core"
	"go-chronos/metrics"
	"go-chronos/rpc/pb"
	"go-chronos/utils"
)

type transactionService struct {
	pb.UnimplementedTransactionServer
}

//var TransactionService = transactionService{}

// SubmitTransaction 交易提交接口
func (s *transactionService) SubmitTransaction(ctx context.Context, in *pb.SubmitTransactionReq) (*pb.SubmitTransactionRsp, error) {
	// 从 req 中获取到已经签名的交易
	resp := new(pb.SubmitTransactionRsp)
	recvTransactionCode := in.GetSignedTransaction()

	bytesTransaction, err := hex.DecodeString(recvTransactionCode)

	// base 16 解码，如果解码错误直接返回
	if err != nil {
		resp.Status = pb.SubmitTransactionStatus_DECODE_FAILED.Enum()
		resp.Error = proto.String("Decode transaction to bytes failed.")
		//log.Infoln("Decode transaction to bytes failed.")
		return resp, err
	}

	// 对字节数据进行反序列化，如果反序列化错误直接返回
	transaction, err := utils.DeserializeTransaction(bytesTransaction)

	if err != nil {
		resp.Status = pb.SubmitTransactionStatus_DESERIALIZE_FAILED.Enum()
		resp.Error = proto.String("Deserialize transaction failed.")
		//log.Infoln("Deserialize transaction failed.")
		return resp, err
	}

	// 对交易的签名进行验证，如果验证错误直接返回
	//if !transaction.Verify() {
	//	resp.Status = pb.SubmitTransactionStatus_SIGNATURE_FAILED.Enum()
	//	resp.Error = proto.String("Verify transaction signature failed.")
	//	//log.Infoln("Verify transaction signature failed.")
	//	return resp, err
	//}

	// 获取交易池实例，然后添加交易
	pool := core.GetTxPoolInst()
	pool.Add(transaction)
	//handler := node.GetHandlerInst()
	//handler.AddTransaction(transaction)

	//log.Infoln("Append transaction successful.")
	resp.Status = pb.SubmitTransactionStatus_SUCCESS.Enum()
	resp.Error = proto.String("Success.")
	metrics.SubmitTxCountsMetricsInc()
	return resp, err
}
