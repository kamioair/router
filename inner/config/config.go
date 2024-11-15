package config

import (
	_ "embed"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
)

// Config 自定义配置
var Config = struct {
	DownMqtt qdefine.BrokerConfig
}{
	DownMqtt: qdefine.BrokerConfig{
		Addr:    "",
		UId:     "",
		Pwd:     "",
		LogMode: "",
		TimeOut: 0,
		Retry:   0,
	},
}

func Init(module string) {
	qconfig.Load(module, &Config)
}
