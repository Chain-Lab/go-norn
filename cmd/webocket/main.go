/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package main

import (
	"github.com/chain-lab/go-chronos/pubsub"
	"net/http"
)

func main() {
	router := pubsub.CreateNewEventRouter()
	http.HandleFunc("/subscribe", router.HandleConnect)
	http.ListenAndServe("localhost:8888", nil)
}
