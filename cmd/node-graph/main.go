/**
  @author: decision
  @date: 2023/9/18
  @note:
**/

package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	graphLock sync.RWMutex
)

func main() {
	LoadConfig("config.yml")

	nodes := config.Strings("service.nodes")
	if len(nodes) <= 0 {
		log.Errorln("Nodes is empty.")
		return
	}

	port := fmt.Sprintf(":%d", config.Int("service.port"))
	interval := time.Duration(config.Int("service.interval")) * time.Second

	go ConnectNodeInfoRoutine(interval, nodes)

	service := gin.Default()
	service.GET("/api/graph/fields", GraphFieldsService)
	service.GET("/api/graph/data", GraphDataService)
	service.GET("/api/health", GraphHealthService)

	log.Infof("Node graph service start on %s", port)
	service.Run(port)

}

func LoadConfig(filepath string) {
	config.WithOptions(config.ParseEnv)

	config.AddDriver(yaml.Driver)

	err := config.LoadFiles(filepath)
	if err != nil {
		log.WithField("error", err).Errorln("Load config file failed.")
	}
}
