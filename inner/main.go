package main

import (
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"router/inner/config"
	"strings"
)

func main() {
	setting := qservice.NewSetting(DefModule, DefDesc, Version).
		BindInitFunc(onInit).
		BindReqFunc(onReqHandler).
		BindNoticeFunc(onNoticeHandler).
		BindRetainNoticeFunc(onRetainNoticeHandler)

	// 配置初始化
	config.Init(setting.Module)

	// 防止冲突，如果没有请求到客户端ID时，先随机生成一个
	code, err := qservice.DeviceCode.LoadFromFile()
	if err != nil {
		code.Id = qdefine.NewUUID()
	}
	switch strings.ToLower(config.Config.Mode) {
	// 顶级模式和服务端模式，不用附加ID，因为有且只有一个
	case "root", "server":
		setting.SetDeviceCode("[none]" + code.Id)
	default:
		setting.SetDeviceCode(code.Id)
	}

	// 启动微服务
	service = qservice.NewService(setting)
	service.Run()
}
