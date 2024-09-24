package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/crypto"
	"github.com/chain-lab/go-norn/node"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"net"
	"time"
)

func NewKDHT(ctx context.Context, host host.Host, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	// fixed(230918): 和 bootstrap 建立连接后不再创建新的流

	// dht 的配置项
	var options []dht.Option

	// 如果没有引导节点，以服务器模式 ModeServer 启动
	//if len(bootstrapPeers) == 0 {
	options = append(options, dht.Mode(dht.ModeServer))
	//log.Infoln("Start node as a bootstrap server.")
	//}

	// 生成一个 DHT 实例
	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err
	}

	// 启动 DHT 服务
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	// 遍历引导节点数组并尝试连接
	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		if err := host.Connect(ctx, *peerinfo); err != nil {
			log.Printf("Error while connecting to node %q: %-v", peerinfo, err)
			continue
		} else {
			log.Infof("Connection established with bootstrap node: %q",
				*peerinfo)
		}
	}

	return kdht, nil
}

func sendTransaction(pm *node.P2PManager) {
	//log.Infof("Start send transactions.")
	prv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	//ticket := time.NewTicker(1 * time.Millisecond)

	for {
		//select {
		//case <-ticket.C:
		tx := buildTransaction(prv)
		pm.AddTransaction(tx)
		//}
	}
}

func buildResourceManager() *network.ResourceManager {
	scalingLimits := rcmgr.DefaultLimits

	libp2p.SetDefaultServiceLimits(&scalingLimits)

	scaledDefaultLimits := scalingLimits.AutoScale()

	cfg := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			//Conns: 20,
			Streams: 50,
		},
	}

	limits := cfg.Build(scaledDefaultLimits)

	limiter := rcmgr.NewFixedLimiter(limits)

	rm, err := rcmgr.NewResourceManager(limiter)

	if err != nil {
		log.Errorln("Build resource manager failed.")
		return nil
	}

	return &rm
}

func buildTransaction(key *ecdsa.PrivateKey) *common.Transaction {
	// 生成随机数据
	data := make([]byte, 32)
	rand.Read(data)
	timestamp := time.Now().UnixMilli()

	// 构建交易体
	ipAddress := getLocalIP()

	txBody := common.TransactionBody{
		Data:      data,
		Timestamp: timestamp,
		Expire:    timestamp + 3000,
		IP:        ipAddress,
	}

	// 设置交易的公钥、地址，初始化哈希值、签名为空
	txBody.Public = [33]byte(crypto.PublicKey2Bytes(&key.PublicKey))
	txBody.Address = crypto.PublicKeyBytes2Address(txBody.Public)
	txBody.Hash = [32]byte{}
	txBody.Signature = []byte{}

	// 将当前未签名的交易进行序列化 -> 字节形式
	writer := karmem.NewWriter(1024)
	txBody.WriteAsRoot(writer)
	txBodyBytes := writer.Bytes()

	// 哈希序列化后的交易，然后签名
	hash := sha256.New()
	hash.Write(txBodyBytes)
	txHashBytes := hash.Sum(nil)
	txSignatureBytes, err := ecdsa.SignASN1(rand.Reader, key, txHashBytes)

	if err != nil {
		log.WithField("error", err).Errorln("Sign transaction failed.")
		return nil
	}

	// 写入签名和哈希信息
	txBody.Hash = [32]byte(txHashBytes)
	txBody.Signature = txSignatureBytes
	tx := common.Transaction{
		Body: txBody,
	}

	return &tx
}
func getLocalIP() [4]byte {
	var ipArray [4]byte
	loopbackIP := net.IPv4(127, 0, 0, 1) // 默认环回地址
	copy(ipArray[:], loopbackIP)
	return ipArray
}
