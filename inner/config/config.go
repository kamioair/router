package config

import (
	_ "embed"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
)

// Config 自定义配置
var Config = struct {
	StartId int
	Mode    ERouteMode           // 路由模式 client/server
	UpMqtt  qdefine.BrokerConfig // 上级Broker配置
}{
	Mode: ERouteClient,
	UpMqtt: qdefine.BrokerConfig{
		Addr:    "",
		UId:     "",
		Pwd:     "",
		LogMode: "NONE",
		TimeOut: 3000,
		Retry:   3,
	},
	StartId: 1000,
}

type ERouteMode string

const (
	ERouteClient ERouteMode = "client"
	ERouteServer ERouteMode = "server"
)

func Init(module string) {
	qconfig.Load(module, &Config)
}
