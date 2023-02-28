package node

import "go-chronos/p2p"

type msgHandler func()

var handlerMap = map[p2p.StatusCode]msgHandler{}
