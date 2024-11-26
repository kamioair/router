package config

import (
	_ "embed"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qconfig"
)

// Config 自定义配置
var Config = struct {
	Mode    ERouteMode           // 路由模式 client/server
	Monitor MonitorConfig        // 监控配置
	UpMqtt  qdefine.BrokerConfig // 上级Broker配置
}{
	Mode: ERouteClient,
	Monitor: MonitorConfig{
		Cron: "0/10 * * * * ?",
		CpuAlarm: CpuAlarm{
			Value:    10,
			Duration: 30,
		},
		MemAlarm:  90,
		DiskAlarm: 90,
		DiskPaths: []string{},
		Processes: []string{},
	},
	UpMqtt: qdefine.BrokerConfig{
		Addr:    "",
		UId:     "",
		Pwd:     "",
		LogMode: "NONE",
		TimeOut: 3000,
		Retry:   3,
	},
}

type MonitorConfig struct {
	Cron      string   // 检测间隔
	CpuAlarm  CpuAlarm // CPU报警值
	MemAlarm  int      // 内存报警值
	DiskAlarm int      // 硬盘报警值
	DiskPaths []string // 需要检测的硬盘分区，不填写默认检测（客户端：系统所在分区/ 服务端：所有分区）
	Processes []string // 需要监控存活的进程名称
}

type CpuAlarm struct {
	Value    float64
	Duration float64
}

type ERouteMode string

const (
	ERouteClient ERouteMode = "client"
	ERouteServer ERouteMode = "server"
)

func Init(module string) {
	qconfig.Load(module, &Config)
}
