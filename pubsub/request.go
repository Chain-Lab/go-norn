/**
  @author: decision
  @date: 2024/1/3
  @note:
**/

package pubsub

import "encoding/json"

type EventRequest struct {
	Address   string `json:"address"`
	EventType string `json:"type"`
}

func DeserializeEventRequest(requestStr []byte) (*EventRequest, error) {
	request := new(EventRequest)
	err := json.Unmarshal(requestStr, request)

	if err != nil {
		return nil, err
	}

	return request, nil
}
