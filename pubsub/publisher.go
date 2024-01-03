/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package pubsub

import (
	"github.com/gorilla/websocket"
	"sync"
)

const (
	connPoolSize = 128
)

type EventPublisher struct {
	conns []*websocket.Conn
	count int

	lock sync.RWMutex
}

func CreateNewEventPublisher() *EventPublisher {
	publisher := EventPublisher{
		conns: make([]*websocket.Conn, connPoolSize),
		count: 0,
	}

	return &publisher
}

func (e *EventPublisher) IsFull() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.count >= connPoolSize
}

func (e *EventPublisher) AppendNewConnection(conn *websocket.Conn) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.conns[e.count] = conn
}
