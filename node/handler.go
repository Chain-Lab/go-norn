package node

type msgHandler func()

var handlerMap = map[uint8]msgHandler{}
