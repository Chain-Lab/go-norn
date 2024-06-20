package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	delta     int64
	bootstrap string
	datadir   string
	cfg       string
	help      bool
	genesis   bool
	debug     bool
	trace     bool
	pp        bool
	metrics   bool
	random    bool
)

func init() {
	// 启动节点的地址信息 -b [bootstrap_address]
	flag.StringVar(&bootstrap, "b", "", "Bootstrap node address")
	// 数据存储目录 -d [path]
	flag.StringVar(&datadir, "d", "./data", "Data directory path")
	// 配置文件 -c [path] 默认为当前目录的 config.yml
	flag.StringVar(&cfg, "c", "./config.yml", "Config file path")
	// 帮助信息 -h
	flag.BoolVar(&help, "h", false, "Command help")
	// 是否创世节点 -g 不添加参数默认为否
	flag.BoolVar(&genesis, "g", false, "Create genesis block after 10s")
	// 日志级别 -debug
	flag.BoolVar(&debug, "debug", false, "Debug log level")
	// 日志级别 -trace
	flag.BoolVar(&trace, "trace", false, "Track log level")
	// 是否启用 pprof 性能分析， 如果是则没每次运行结束会生成 cpu、内存相关的信息
	flag.BoolVar(&pp, "pprof", false, "Track with pprof")
	// 是否启用指标监控，如果启用选项则会开放端口用于指标监控
	flag.BoolVar(&metrics, "metrics", false, "Open metrics service")
	// 是否随机启动，如果否，则会在 10s 后启动
	flag.BoolVar(&random, "random", false, "Random time start")
	flag.Int64Var(&delta, "delta", 0, "Initial time delta (for test)")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `chronos version: 1.0.0
Usage: chronos [-b bootstrap] [-d datadir] [-c config] [-h help] [-g genesis] [--debug]

Options:
`)
	flag.PrintDefaults()
}
