package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	bootstrap string
	datadir   string
	port      int
	help      bool
	genesis   bool
	debug     bool
	test      bool
	trace     bool
)

func init() {
	flag.StringVar(&bootstrap, "b", "", "Bootstrap node address")
	flag.StringVar(&datadir, "d", "./data", "Data directory path")
	flag.IntVar(&port, "p", 31258, "Listening port")
	flag.BoolVar(&help, "h", false, "Command help")
	flag.BoolVar(&genesis, "g", false, "Create genesis block after 10s")
	flag.BoolVar(&debug, "debug", false, "Debug log level")
	flag.BoolVar(&test, "test", false, "Send transaction test")
	flag.BoolVar(&trace, "trace", false, " Track log level")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `chronos version: 1.0.0
Usage: chronos [-b bootstrap] [-d datadir] [-p port] [-h help] [-g genesis] [--debug]

Options:
`)
	flag.PrintDefaults()
}
