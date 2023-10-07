/**
  @author: decision
  @date: 2023/9/18
  @note:
**/

package main

import (
	"context"
	"github.com/chain-lab/go-chronos/rpc/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

type Node struct {
	Id       int    `json:"id"`
	Identity string `json:"identity"`
	Title    string `json:"title"`
}

type Edge struct {
	Id     int `json:"id"`
	Source int `json:"source"`
	Target int `json:"target"`
}

var (
	edges []*Edge
	nodes []*Node
)

func ConnectNodeInfoRoutine(interval time.Duration, nodesAddr []string) {
	nodes = make([]*Node, 0, 100)
	edges = make([]*Edge, 0)
	nodesMap := make(map[string]*Node)
	count := 1

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			edgesResult := make([]*Edge, 0, 200)
			edgesCount := 1
			for _, addr := range nodesAddr {
				conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					log.Debug("Connect to node %s failed.", addr)
				}

				c := pb.NewNodeClient(conn)
				resp, err := c.ConnectedNodeList(context.Background(), &pb.ConnectedNodeReq{})

				if resp.GetCode() != pb.NodeStatusRespCodes_NODE_STATUS_SUCCESS {
					log.Debugf("Get remote node %s connected info failed.",
						addr)
					continue
				}

				hostId := *resp.Local
				node, ok := nodesMap[hostId]
				if !ok {
					n := &Node{
						Id:       count,
						Identity: hostId,
						Title:    hostId[:5],
					}
					nodes = append(nodes, n)
					nodesMap[hostId] = n
					node = n
					count++
				}

				for _, remoteId := range resp.Remote {
					remote, ok := nodesMap[remoteId]
					if !ok {
						n := &Node{
							Id:       count,
							Identity: remoteId,
							Title:    remoteId[:5],
						}
						nodesMap[remoteId] = n
						remote = n
						nodes = append(nodes, n)
						count++
					}

					edgesResult = append(edgesResult, &Edge{
						Id:     edgesCount,
						Source: node.Id,
						Target: remote.Id,
					})
					edgesCount++
				}
			}
			graphLock.Lock()
			if len(edgesResult) != 0 {
				edges = edgesResult
			}
			graphLock.Unlock()
		}
	}
}
