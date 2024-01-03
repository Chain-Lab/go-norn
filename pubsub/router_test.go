/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package pubsub

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"testing"
)

func TestEventRouter_HandleConnect(t *testing.T) {
	router := CreateNewEventRouter()
	http.HandleFunc("/subscribe", router.HandleConnect)
	log.Fatal(http.ListenAndServe("localhost:8888", nil))
}
