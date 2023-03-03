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
)

func init() {
	flag.StringVar(&bootstrap, "b", "", "Bootstrap node address")
	flag.StringVar(&datadir, "d", "./data", "Data directory path")
	flag.IntVar(&port, "p", 31258, "Listening port")
	flag.BoolVar(&help, "h", false, "Command help")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `chronos version: 1.0.0
Usage: chronos [-b bootstrap] [-d datadir] [-p port] [-h help]

Options:
`)
	flag.PrintDefaults()
}
