package config

import (
	_ "embed"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
)

// Config 自定义配置
var Config = struct {
	Mode   ERouteMode           // 路由模式 client/server
	UpMqtt qdefine.BrokerConfig // 上级Broker配置
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
}

// Monitor 监控配置
var Monitor = struct {
	Cron      string   // 检测间隔
	CpuAlarm  float64  // CPU报警值
	MemAlarm  float64  // 内存报警值
	DiskAlarm float64  // 硬盘报警值
	Duration  float64  // 达到报警值的持续时间
	DiskPaths []string // 需要检测的硬盘分区，不填写默认检测（客户端：系统所在分区/ 服务端：所有分区）
	Processes []string // 需要监控存活的进程名称
}{
	Cron:      "0/10 * * * * ?",
	CpuAlarm:  95,
	MemAlarm:  95,
	DiskAlarm: 95,
	Duration:  30,
	DiskPaths: []string{"C:", "D:"},
	Processes: []string{},
}

type ERouteMode string

const (
	ERouteClient ERouteMode = "client"
	ERouteServer ERouteMode = "server"
)

func Init(module string) {
	qconfig.Load(module, &Config)
	qconfig.Load("monitor", &Monitor)
}
