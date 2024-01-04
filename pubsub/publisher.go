/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package pubsub

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
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

func (e *EventPublisher) Full() bool {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.count >= connPoolSize
}

func (e *EventPublisher) Publish(data []byte) {
	e.lock.Lock()
	defer e.lock.Unlock()

	for idx, conn := range e.conns {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.WithError(err).Errorln("Write message to conn failed.")
			e.conns[idx] = nil
			e.count--
			continue
		}
	}
}

func (e *EventPublisher) AppendNewConnection(conn *websocket.Conn) {
	e.lock.Lock()
	defer e.lock.Unlock()

	pos := e.SelectPosition()
	if pos != -1 {
		e.conns[pos] = conn
		e.count++
	}
}

func (e *EventPublisher) SelectPosition() int {
	for idx := range e.conns {
		if e.conns[idx] == nil {
			return idx
		}
	}

	return -1
}
