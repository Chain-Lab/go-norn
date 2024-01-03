/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package pubsub

import (
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

var (
	routerOnce = sync.Once{}
	routerInst *EventRouter
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type EventTopic string

type EventRouter struct {
	publishMap map[EventTopic]*EventPublisher
	//publishChan chan

	lock sync.RWMutex
}

func CreateNewEventRouter() *EventRouter {
	routerOnce.Do(func() {
		routerInst = &EventRouter{
			publishMap: make(map[EventTopic]*EventPublisher),
		}
	})

	return routerInst
}

func (e *EventRouter) HandleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Errorln("Handle event websocket request failed.")
		return
	}

	_, message, err := conn.ReadMessage()
	log.Infof("Receive message from websocket: %s", message)

	request, err := DeserializeEventRequest(message)

	if err != nil {
		log.WithError(err).Debugln("Deserialize request to object failed.")
		return
	}

	if request.Address == "" || request.EventType == "" {
		log.Debugln("Address or type is empty...")
		return
	}

	topic := toTopicStr(request.Address, request.EventType)

	e.lock.Lock()
	defer e.lock.Unlock()
	publisher, ok := e.publishMap[topic]

	if !ok || publisher == nil {
		e.publishMap[topic] = CreateNewEventPublisher()
	}
	publisher, _ = e.publishMap[topic]

	if publisher.IsFull() {
		log.Debugln("Publisher connection is full.")
		conn.Close()
		return
	}

	publisher.AppendNewConnection(conn)
}

func toTopicStr(address string, eventType string) EventTopic {
	strTopic := fmt.Sprintf("%s#%s", address,
		eventType)

	return EventTopic(strTopic)
}
