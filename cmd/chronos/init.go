package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	bootstrap string
	datadir   string
	cfg       string
	help      bool
	genesis   bool
	debug     bool
	trace     bool
	pp        bool
)

func init() {
	flag.StringVar(&bootstrap, "b", "", "Bootstrap node address")
	flag.StringVar(&datadir, "d", "./data", "Data directory path")
	flag.StringVar(&cfg, "c", "./config.yml", "Config file path")
	flag.BoolVar(&help, "h", false, "Command help")
	flag.BoolVar(&genesis, "g", false, "Create genesis block after 10s")
	flag.BoolVar(&debug, "debug", false, "Debug log level")
	flag.BoolVar(&trace, "trace", false, "Track log level")
	flag.BoolVar(&pp, "pprof", false, "Track with pprof")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `chronos version: 1.0.0
Usage: chronos [-b bootstrap] [-d datadir] [-p port] [-h help] [-g genesis] [--debug]

Options:
`)
	flag.PrintDefaults()
}
