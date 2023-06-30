package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gookit/config/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go-chronos/core"
	"go-chronos/node"
	"go-chronos/rpc"
	"go-chronos/utils"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"
)

// 测试指令：
// ./chronos -d ./data1 -g -c config1.yml
// ./chronos -d ./data2 -c config2.yml --metrics -b /ip4/127.0.0.1/tcp/31258/p2p/12D3KooWJtvSD3yzu1XpKxr3eKutgjJXgky266AdnUJSg25ZXuVr
// ./chronos -d ./data2 -c config2.yml --metrics --delta 40000 -b /ip4/127.0.0.1/tcp/31258/p2p/12D3KooWJtvSD3yzu1XpKxr3eKutgjJXgky266AdnUJSg25ZXuVr
// arm64： CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o chronos_arm64
// pprof 性能分析：
// go tool pprof -http=:8080 cpu.profile
func main() {
	flag.Parse()

	var f *os.File
	if pp {
		fileName := fmt.Sprintf("cpu-%d.profile", time.Now().UnixMilli())
		f, _ := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
		//defer f.Close()
		pprof.StartCPUProfile(f)
		//defer pprof.StopCPUProfile()
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
		go http.ListenAndServe(metricPort, nil)
		log.Infof("Metric server start on localhost%s", metricPort)
	}

	// RPC 协程服务开启
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
	txPool := core.GetTxPoolInst()
	hConfig := node.HandlerConfig{
		TxPool:       txPool,
		Chain:        chain,
		Genesis:      genesis,
		InitialDelta: delta,
	}

	h, err := node.NewHandler(&hConfig)
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

	host, err := libp2p.New(
		libp2p.ListenAddrs(localMultiAddr),
	)
	if err != nil {
		log.WithField("error", err).Errorln("Create local host failed.")
	}

	host.SetStreamHandler(node.ProtocolId, node.HandleStream)
	// 打印节点的 id 信息
	//log.Infof("Node address: /ip4/127.0.0.1/tcp/%v/p2p/%s", port, host.ID().String())
	log.Infof("Node address: /ip4/192.168.31.119/tcp/%v/p2p/%s", port, host.ID().String())

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
			log.WithField("error", err).Errorln("Covert address to multiple addrerss failed.")
			return
		}
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{maddr})
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	}

	// 节点发现协程
	go node.Discover(ctx, host, kdht, "Chronos network.")

	if genesis {
		log.Infof("Create genesis block after 10s...")
		go func() {
			ticker := time.NewTicker(10 * time.Second)

			select {
			case <-ticker.C:
				log.Infof("Create genesis block.")
				chain.NewGenesisBlock()

				// 创建创世区块时默认已经完成同步
				// todo：这里存在一个问题，如果在未同步时添加创世区块选项，会默认设置完成同步
				//  所以还需要检查创世区块的创建状态
				h.SetSynced()
			}
		}()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	select {
	case sign := <-c:
		log.Infof("Got %s signal. Aborting...", sign)

		if pp {
			pprof.StopCPUProfile()
			f.Close()
		}

		os.Exit(1)
	}
}
