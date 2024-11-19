package config

import (
	_ "embed"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
)

// Config 自定义配置
var Config = struct {
	Mode    string               // 路由模式 空/server/root
	UpMqtt  qdefine.BrokerConfig // 上级Broker配置
	StartId int
}{
	Mode: "",
	UpMqtt: qdefine.BrokerConfig{
		Addr:    "",
		UId:     "",
		Pwd:     "",
		LogMode: "",
		TimeOut: 0,
		Retry:   0,
	},
	StartId: 1000,
}

func Init(module string) {
	qconfig.Load(module, &Config)
}
