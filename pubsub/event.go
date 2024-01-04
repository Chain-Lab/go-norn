/**
  @author: decision
  @date: 2024/1/4
  @note:
**/

package pubsub

type Event struct {
	Type    string            `json:"type"`
	Hash    string            `json:"hash"`
	Height  string            `json:"height"`
	Address string            `json:"address"`
	Params  map[string]string `json:"params"`
}
