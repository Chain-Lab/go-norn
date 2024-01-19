/**
  @author: decision
  @date: 2023/12/21
  @note:
**/

package core

import (
	"encoding/hex"
	"encoding/json"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/pubsub"
	"github.com/chain-lab/go-chronos/utils"
	log "github.com/sirupsen/logrus"
	"strconv"
)

const (
	taskChannelSize int = 10240
)

var (
	emptyMap = map[string]string{}
)

type DataTask struct {
	Type    string
	Hash    common.Hash
	Height  int64
	Address []byte
	Key     []byte
	Value   []byte
}

type DataProcessor struct {
	taskChannel chan *DataTask
	db          *utils.LevelDB
}

func NewDataProcessor() *DataProcessor {
	taskChan := make(chan *DataTask, taskChannelSize)

	return &DataProcessor{
		taskChannel: taskChan,
	}
}

func (dp *DataProcessor) Run() {
	for {
		select {
		// todo: batch insert with a buffer
		case task := <-dp.taskChannel:
			if task.Type == setCommandString {
				dp.setData(task)
			} else if task.Type == appendCommandString {
				taskValue := task.Value
				mapValue := map[string]string{}
				err := json.Unmarshal(taskValue, &mapValue)

				if err != nil {
					log.WithError(err).Errorln("Receive append task failed.")
					continue
				}
				dp.appendDate(task)
			}
		}
	}
}

func (dp *DataProcessor) setData(task *DataTask) {
	db := dp.db
	dbKey := utils.DataAddressKey2DBKey(task.Address, task.Key)

	mapValue := map[string]string{}
	var mapArray []map[string]string
	var value []byte

	err := json.Unmarshal(task.Value, &mapValue)
	if err != nil {
		value = task.Value
	} else {
		mapArray = append(mapArray, mapValue)
		value, _ = json.Marshal(mapArray)
	}
	_ = db.Insert(dbKey, task.Value)
	log.Infof("Trying insert data with key %s and value %s", dbKey,
		string(value))

	params := map[string]string{
		"key":   string(task.Key),
		"value": string(value),
	}

	router := pubsub.CreateNewEventRouter()
	event := pubsub.Event{
		Type:    "data",
		Hash:    hex.EncodeToString(task.Hash[:]),
		Height:  strconv.Itoa(int(task.Height)),
		Address: hex.EncodeToString(task.Address),
		Params:  params,
	}

	log.Infof("Append task to router.")
	router.AppendEvent(event)
}

func (dp *DataProcessor) appendDate(task *DataTask) {
	db := dp.db

	dbKey := utils.DataAddressKey2DBKey(task.Address, task.Key)

	mapValue := map[string]string{}
	var mapArray []map[string]string
	err := json.Unmarshal(task.Value, &mapValue)
	if err != nil {
		log.WithError(err).Errorln("task value is not map")
		return
	}
	value, err := db.Get(dbKey)
	log.Infoln(string(value))
	if err == nil {
		log.Info(string(value))
		err = json.Unmarshal(value, &mapArray)
		if err != nil {
			log.WithError(err).Errorln("database value is not map array")
			return
		}
	}
	mapArray = append(mapArray, mapValue)

	value, err = json.Marshal(mapArray)
	if err != nil {
		log.WithError(err).Errorln("Marshal json array failed.")
		return
	}

	_ = db.Insert(dbKey, value)
	log.Infof("Trying append data with key %s and value %s", dbKey,
		string(value))

	params := map[string]string{
		"key":   string(task.Key),
		"value": string(value),
	}

	router := pubsub.CreateNewEventRouter()
	event := pubsub.Event{
		Type:    "data",
		Hash:    hex.EncodeToString(task.Hash[:]),
		Height:  strconv.Itoa(int(task.Height)),
		Address: hex.EncodeToString(task.Address),
		Params:  params,
	}

	log.Infof("Append task to router.")
	router.AppendEvent(event)
}
