/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package pubsub

import (
	"encoding/json"
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
	publishMap  map[EventTopic]*EventPublisher
	publishChan chan Event

	lock sync.RWMutex
}

func CreateNewEventRouter() *EventRouter {
	routerOnce.Do(func() {
		routerInst = &EventRouter{
			publishMap:  make(map[EventTopic]*EventPublisher),
			publishChan: make(chan Event, 256),
		}
	})

	return routerInst
}

func (e *EventRouter) AppendEvent(event Event) {
	e.publishChan <- event
}

func (e *EventRouter) Process() {
	for {
		select {
		case event := <-e.publishChan:
			topic := toTopicStr(event.Address, event.Type)
			log.Infof("Receive task with topic %s", topic)
			e.lock.RLock()

			publisher, ok := e.publishMap[topic]
			if publisher == nil || !ok {
				e.lock.RUnlock()
				continue
			}

			eventData, err := json.Marshal(event)
			if err != nil {
				log.WithError(err).Debugln("Marshal event to bytes failed.")
				e.lock.RUnlock()
				continue
			}

			publisher.Publish(eventData)

			e.lock.RUnlock()
		}
	}
}

func (e *EventRouter) HandleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Errorln("Handle event websocket request failed.")
		conn.Close()
		return
	}

	_, message, err := conn.ReadMessage()
	log.Infof("Receive message from websocket: %s", message)

	request, err := DeserializeEventRequest(message)

	if err != nil {
		log.WithError(err).Errorln("Deserialize request to object failed.")
		conn.Close()
		return
	}

	if request.Address == "" || request.EventType == "" {
		log.Errorln("Address or type is empty...")
		conn.Close()
		return
	}

	topic := toTopicStr(request.Address, request.EventType)
	log.Infof("Receive subscribe with topic %s", topic)

	e.lock.Lock()
	defer e.lock.Unlock()
	publisher, ok := e.publishMap[topic]

	if !ok || publisher == nil {
		e.publishMap[topic] = CreateNewEventPublisher()
	}
	publisher, _ = e.publishMap[topic]

	if publisher.Full() {
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
