package main

import (
	"github.com/kamioair/qf/qservice"
	"router/inner/config"
)

func main() {
	setting := qservice.NewSetting(DefModule, DefDesc, Version).
		BindInitFunc(onInit).
		BindReqFunc(onReqHandler).
		BindNoticeFunc(onNoticeHandler).
		BindStatusFunc(onStatusHandler)

	// 配置初始化
	config.Init(setting.Module, setting.Mode)

	// 设置设备ID
	setting.DevCode = config.DeviceId()

	// 启动微服务
	service = qservice.NewService(setting)
	service.Run()
}
