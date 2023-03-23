package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/crypto"
	"go-chronos/node"
	karmem "karmem.org/golang"
	"time"
)

func NewKDHT(ctx context.Context, host host.Host, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	// dht 的配置项
	var options []dht.Option

	// 如果没有引导节点，以服务器模式 ModeServer 启动
	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
		log.Infoln("Start node as a bootstrap server.")
	}

	// 生成一个 DHT 实例
	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err
	}

	// 启动 DHT 服务
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	h := node.GetHandlerInst()

	// 遍历引导节点数组并尝试连接
	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		if err := host.Connect(ctx, *peerinfo); err != nil {
			log.Printf("Error while connecting to node %q: %-v", peerinfo, err)
		} else {
			s, err := host.NewStream(ctx, peerinfo.ID, node.ProtocolId)

			if err != nil {
				log.WithField("error", err).Errorln("Create new stream error.")
			}
			_, err = h.NewPeer(peerinfo.ID, &s)
			log.Printf("Connection established with bootstrap node: %q", *peerinfo)
		}
	}

	return kdht, nil
}

func sendTransaction(h *node.Handler) {
	prv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	//ticket := time.NewTicker(50 * time.Millisecond)

	for {
		//select {
		//case <-ticket.C:
		tx := buildTransaction(prv)
		h.AddTransaction(tx)
		//}
	}
}

func buildTransaction(key *ecdsa.PrivateKey) *common.Transaction {
	data := make([]byte, 32)
	rand.Read(data)
	timestamp := time.Now().UnixMilli()

	txBody := common.TransactionBody{
		Data:      data,
		Timestamp: timestamp,
		Expire:    timestamp + 3000,
	}

	txBody.Public = [33]byte(crypto.PublicKey2Bytes(&key.PublicKey))
	txBody.Address = crypto.PublicKeyBytes2Address(txBody.Public)
	txBody.Hash = [32]byte{}
	txBody.Signature = []byte{}

	writer := karmem.NewWriter(1024)
	txBody.WriteAsRoot(writer)
	txBodyBytes := writer.Bytes()

	hash := sha256.New()
	hash.Write(txBodyBytes)
	txHashBytes := hash.Sum(nil)
	txSignatureBytes, err := ecdsa.SignASN1(rand.Reader, key, txHashBytes)

	if err != nil {
		log.WithField("error", err).Errorln("Sign transaction failed.")
		return nil
	}

	txBody.Hash = [32]byte(txHashBytes)
	txBody.Signature = txSignatureBytes
	tx := common.Transaction{
		Body: txBody,
	}

	return &tx
}
