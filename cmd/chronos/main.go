package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/chain-lab/go-chronos/core"
	metrics2 "github.com/chain-lab/go-chronos/metrics"
	"github.com/chain-lab/go-chronos/node"
	"github.com/chain-lab/go-chronos/pubsub"
	"github.com/chain-lab/go-chronos/rpc"
	"github.com/chain-lab/go-chronos/utils"
	"github.com/gookit/config/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"
)

// 测试指令：
// ./chronos -d ./data1 -g -c config1.yml
// ./chronos -d ./data -g --metrics --pprof -c config.yml
// ./chronos -d ./data -c config.yml --metrics -b /ip4/43.134.123.140/tcp/31258/p2p/QmNuqv3q7kzxtquzbnDEYLuNmPrwB2G1ZHRmzEH6dTFFbS
// nohup ./chronos -d ./data -c config.yml --metrics -b /ip4/43.134.29.89/tcp/31258/p2p/QmcCGvGWyACcyadfXmXoYw6E8WjdfnBvvotyh3cFNTfTCA >> output 2>&1 &
// ./chronos -d ./data2 -c config2.yml --metrics --delta 40000 -b /ip4/127.0.0.1/tcp/31258/p2p/12D3KooWJtvSD3yzu1XpKxr3eKutgjJXgky266AdnUJSg25ZXuVr
// ./chronos -d ./data2 -c config2.yml --metrics --pprof -b /ip4/127.0.0.1/tcp/31258/p2p/QmYwdCNHr1fKyURJWC6Pi5889ei6gm3kL9VczVfgxPRXgi
// ./chronos -d ./data3 -c config3.yml --metrics --pprof -b /ip4/127.0.0.1/tcp/31258/p2p/QmYwdCNHr1fKyURJWC6Pi5889ei6gm3kL9VczVfgxPRXgi
// ./chronos -d ./data -c config.yml --metrics -b /ip4/172.22.32.5/tcp/31258/p2p/QmZzEG4tpnMwJLWWU71EFCfp1vYWCaRCkjNejetbjgvQtu
// ./chronos -d ./data2 -c config2.yml --metrics --pprof -b
// arm64： CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o chronos_arm64
// amd64： CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o chronos_amd64
// pprof 性能分析：
// go tool pprof -http=:8080 cpu.profile
func main() {
	flag.Parse()

	var f *os.File

	if random {
		r := rand.New(rand.NewSource(time.Now().UnixMilli()))
		sleepTime := r.Intn(120)
		log.Infof("Random sleep for %d s", sleepTime)
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}

	if pp {
		fileName := fmt.Sprintf("cpu-%d.profile", time.Now().UnixMilli())
		f, _ := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
		pprof.StartCPUProfile(f)
		go http.ListenAndServe(":6060", nil)
	}

	// 显示帮助信息，每个选项相关的功能
	if help {
		flag.Usage()
		return
	}

	// 当前的日志级别是否设置为 Trace
	if trace {
		log.SetLevel(log.TraceLevel)
	}

	// 当前的日志级别是否设置为 Debug
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// 加载 config 配置文件
	core.LoadConfig(cfg)

	//metrics2.RegisterMetrics()
	if metrics {
		metricPort := ":" + config.String("metrics.port")
		http.Handle("/metrics", promhttp.Handler())
		metrics2.RoutineCreateCounterObserve(0)
		go metrics2.RegularMetricsRoutine()
		go http.ListenAndServe(metricPort, nil)
		log.Infof("Metric server start on localhost%s", metricPort)
	}

	// RPC 协程服务开启
	metrics2.RoutineCreateCounterObserve(2)
	go rpc.RPCServerStart()
	port := config.Int("node.port")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 数据库、节点的启动
	db, err := utils.NewLevelDB(datadir)

	if err != nil {
		log.WithField("error", err).Errorln("Create or load database failed.")
		return
	}

	chain := core.NewBlockchain(db)
	txPool := core.NewTxPool(chain)
	hConfig := node.P2PManagerConfig{
		TxPool:       txPool,
		Chain:        chain,
		Genesis:      genesis,
		InitialDelta: delta,
	}

	pm, err := node.NewP2PManager(&hConfig)
	if err != nil {
		log.WithField("error", err).Errorln("Create handler failed.")
		return
	}

	// 网络部分的启动
	localMultiAddr, err := multiaddr.NewMultiaddr(
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
	)
	if err != nil {
		log.WithField("error", err).Errorln("Convert multiple address failed")
		return
	}

	privBytes, err := hex.DecodeString(config.String("p2p.prv"))

	if err != nil {
		log.WithError(err).Errorln("Decode private key failed.")
		return
	}

	identity, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		log.WithError(err).Errorln("Unmarshal private key failed.")
		return
	}

	//rm := buildResourceManager()
	//if rm == nil {
	//	return
	//}

	host, err := libp2p.New(
		libp2p.ListenAddrs(localMultiAddr),
		libp2p.Identity(identity),
		//libp2p.ResourceManager(*rm),
	)
	if err != nil {
		log.WithField("error", err).Errorln("Create local host failed.")
	}

	host.SetStreamHandler(node.ProtocolId, pm.HandleStream)

	// 打印节点的 id 信息
	log.Infof("Node address: /ip4/127.0.0.1/tcp/%v/p2p/%s", port, host.ID().String())

	var kdht *dht.IpfsDHT

	if bootstrap == "" {
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{})
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	} else {
		maddr, err := multiaddr.NewMultiaddr(bootstrap)

		if err != nil {
			log.WithField("error", err).Errorln("Covert address to multiple address failed.")
			return
		}
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{maddr})
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	}

	// 节点发现协程
	metrics2.RoutineCreateCounterObserve(3)
	go pm.Discover(ctx, host, kdht, node.NetworkRendezvous)

	// 事件订阅/发布协程
	router := pubsub.CreateNewEventRouter()
	http.HandleFunc("/subscribe", router.HandleConnect)
	go router.Process()
	go http.ListenAndServe(":8888", nil)

	if genesis {
		log.Infof("Create genesis block after 10s...")
		metrics2.RoutineCreateCounterObserve(4)
		go func() {
			ticker := time.NewTicker(10 * time.Second)

			// 在 10s 后创建创世区块
			select {
			case <-ticker.C:
				log.Infof("Create genesis block.")
				chain.NewGenesisBlock()

				// 创建创世区块时默认已经完成同步
				// todo：这里存在一个问题，如果在未同步时添加创世区块选项，会默认设置完成同步
				//  所以还需要检查创世区块的创建状态
				pm.SetSynced()
			}
		}()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	select {
	case sign := <-c:
		log.Infof("Got %s signal. Aborting...", sign)

		if pp {
			memFileName := fmt.Sprintf("mem-%d.profile", time.Now().UnixMilli())
			memf, _ := os.OpenFile(memFileName, os.O_CREATE|os.O_RDWR, 0644)
			pprof.WriteHeapProfile(memf)

			pprof.StopCPUProfile()
			f.Close()
		}

		os.Exit(1)
	}
}
