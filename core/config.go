/**
  @author: decision
  @date: 2023/5/18
  @note:
**/

package core

import (
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	log "github.com/sirupsen/logrus"
)

// LoadConfig 在启动时运行一次，加载配置文件
func LoadConfig() {
	config.WithOptions(config.ParseEnv)

	config.AddDriver(yaml.Driver)

	err := config.LoadFiles("./config.yaml")
	if err != nil {
		log.WithField("error", err)
	}
}
