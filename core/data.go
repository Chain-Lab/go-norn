/**
  @author: decision
  @date: 2023/12/21
  @note:
**/

package core

import (
	"encoding/hex"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/pubsub"
	"github.com/chain-lab/go-chronos/utils"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const (
	taskChannelSize int = 10240
)

type DataTask struct {
	hash    common.Hash
	height  int64
	address []byte
	key     []byte
	value   []byte
}

type DataProcessor struct {
	taskChannel chan DataTask
	db          *utils.LevelDB
}

func NewDataProcessor() *DataProcessor {
	taskChan := make(chan DataTask, taskChannelSize)

	return &DataProcessor{
		taskChannel: taskChan,
	}
}

func (dp *DataProcessor) Run() {
	db := dp.db

	for {
		select {
		// todo: batch insert with a buffer
		case task := <-dp.taskChannel:
			dbKey := utils.DataAddressKey2DBKey(task.address, task.key)
			log.Infof("Trying insert data with key %s and value %s", dbKey,
				task.value)
			_ = db.Insert(dbKey, task.value)

			params := map[string]string{
				"key":   string(task.key),
				"value": string(task.value),
			}

			router := pubsub.CreateNewEventRouter()
			event := pubsub.Event{
				Type:    "data",
				Hash:    hex.EncodeToString(task.hash[:]),
				Height:  strconv.Itoa(int(task.height)),
				Address: hex.EncodeToString(task.address),
				Params:  params,
			}

			log.Infof("Append task to router.")
			router.AppendEvent(event)
		}
	}
}
