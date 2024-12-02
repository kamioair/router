package config

import (
	_ "embed"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qconfig"
)

// Mode 服务模式
var Mode qservice.EServerMode

// UpMqtt 向上路由配置
var UpMqtt = qdefine.BrokerConfig{
	Addr:    "",
	UId:     "",
	Pwd:     "",
	LogMode: "NONE",
	TimeOut: 3000,
	Retry:   3,
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

func Init(module string, mode qservice.EServerMode) {
	qconfig.Load(module+".upMqtt", &UpMqtt)
	qconfig.Load("monitor", &Monitor)
	Mode = mode

	// 加载设备ID
	device.loadFromFile(mode)
}
