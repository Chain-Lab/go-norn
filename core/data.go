// Package core
// @Description: v1.1.0 新增的数据处理功能，可以在 data 字段写入 set、append 类的指令来添加数据
package core

import (
	"encoding/hex"
	"encoding/json"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/pubsub"
	"github.com/chain-lab/go-norn/utils"
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
	Type    string      // 命令类型，当前有 set、append
	Hash    common.Hash // 该指令对应交易的哈希值
	Height  int64       // 该指令处理的高度
	Address []byte      // 存放、添加数据的地址
	Key     []byte      // 数据的 key
	Value   []byte      // 数据的 value
}

type DataProcessor struct {
	taskChannel chan *DataTask // processor 的任务接收队列
	db          *utils.LevelDB // 数据库实例
}

// NewDataProcessor
//
//	@Description: 实例化数据处理器，接收 channel 传来的任务并进行处理
//	@return *DataProcessor - 数据处理器实例
func NewDataProcessor() *DataProcessor {
	taskChan := make(chan *DataTask, taskChannelSize)

	return &DataProcessor{
		taskChannel: taskChan,
	}
}

// Run
//
//	@Description: 数据处理器的运行函数
//	@receiver DataProcessor 实例
func (dp *DataProcessor) Run() {
	// 无限循环处理任务
	for {
		select {
		// todo: batch insert with a buffer
		// 如果 channel 中存在任务待处理，则取出
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
				dp.appendData(task)
			}
		}
	}
}

// setData
//
//	@Description: 设置数据任务，它会覆盖相同 address、 key 下的数据
//	@receiver DataProcessor 实例
//	@param task - 实例化的任务信息
func (dp *DataProcessor) setData(task *DataTask) {
	db := dp.db
	dbKey := utils.DataAddressKey2DBKey(task.Address, task.Key)

	mapValue := map[string]string{}
	var mapArray []map[string]string
	var value []byte

	// 尝试将任务中的 value 解析为 string -> string 的 json 格式
	// 这里是为了判断 value 是否为一个 {"key1": "value1", ...} 的 json
	// 用来兼容 append 的逻辑
	err := json.Unmarshal(task.Value, &mapValue)
	if err != nil {
		// 如果出错，则解析失败，将 value 直接进行设置
		value = task.Value
	} else {
		// 否则，在一个 [] json 体中追加当前的 json 数据
		mapArray = append(mapArray, mapValue)
		value, _ = json.Marshal(mapArray)
	}
	// 尝试添加到数据库
	_ = db.Insert(dbKey, task.Value)
	log.Infof("Trying insert data with key %s and value %s", dbKey,
		string(value))

	params := map[string]string{
		"key":   string(task.Key),
		"value": string(value),
	}

	// 触发数据变更事件
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

// appendData
//
//	@Description: 添加数据任务，它会在数据后追加数据，而不是覆盖
//	@receiver DataProcessor 实例
//	@param task
func (dp *DataProcessor) appendData(task *DataTask) {
	db := dp.db

	dbKey := utils.DataAddressKey2DBKey(task.Address, task.Key)

	// 这里的代码作用同 setData 函数
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
		// 从数据库中得到 value 并且将它解析为 [{}, {}....] 的形式
		// 如果解析失败则直接触发错误
		log.Info(string(value))
		err = json.Unmarshal(value, &mapArray)
		if err != nil {
			log.WithError(err).Errorln("database value is not map array")
			return
		}
	}
	// 解析成功，向后追加数据
	mapArray = append(mapArray, mapValue)

	value, err = json.Marshal(mapArray)
	if err != nil {
		log.WithError(err).Errorln("Marshal json array failed.")
		return
	}

	// 向数据库中 insert 数据
	_ = db.Insert(dbKey, value)
	log.Infof("Trying append data with key %s and value %s", dbKey,
		string(value))

	params := map[string]string{
		"key":   string(task.Key),
		"value": string(value),
	}

	// 触发数据变更事件
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
