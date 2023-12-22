/**
  @author: decision
  @date: 2023/12/21
  @note:
**/

package core

import (
	"github.com/chain-lab/go-chronos/utils"
	log "github.com/sirupsen/logrus"
)

const (
	taskChannelSize int = 10240
)

type DataTask struct {
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
		}
	}
}
