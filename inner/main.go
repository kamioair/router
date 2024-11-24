package main

import (
	"github.com/kamioair/qf/qdefine"
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
	config.Init(setting.Module)

	// 注册设备ID
	id := ""
	if config.Config.Mode != config.ERouteServer {
		code, err := qservice.DeviceCode.LoadFromFile()
		if err != nil {
			// 当前客户端尚未请求ID，先随机生成一个，防止冲突
			id = qdefine.NewUUID()
		} else {
			id = code.Id
		}
		setting.SetDeviceCode(id)
	}

	// 启动微服务
	service = qservice.NewService(setting)
	service.Run()
}
