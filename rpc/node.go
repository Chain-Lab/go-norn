/**
  @author: decision
  @date: 2023/9/18
  @note:
**/

package rpc

import (
	"context"
	"github.com/chain-lab/go-norn/node"
	"github.com/chain-lab/go-norn/rpc/pb"
)

type nodeService struct {
	pb.UnimplementedNodeServer
}

func (s *nodeService) ConnectedNodeList(ctx context.Context,
	in *pb.ConnectedNodeReq) (*pb.ConnectedNodeResp, error) {
	resp := new(pb.ConnectedNodeResp)
	pm := node.GetP2PManager()

	local, remotes := pm.GetConnectNodeInfo()
	resp.Code = pb.NodeStatusRespCodes_NODE_STATUS_SUCCESS.Enum()

	resp.Local = &local
	resp.Remote = remotes

	return resp, nil
}
