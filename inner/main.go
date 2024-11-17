package main

import (
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
)

func main() {
	// 防止冲突，如果没有请求到客户端ID时，先随机生成一个
	code, err := qservice.DeviceCode.LoadFromFile()
	if err != nil {
		code.Id = qdefine.NewUUID()
	}
	setting := qservice.NewSetting(DefModule, DefDesc, Version).
		BindInitFunc(onInit).
		BindReqFunc(onReqHandler).
		BindNoticeFunc(onNoticeHandler).
		BindRetainNoticeFunc(onRetainNoticeHandler).
		SetDeviceCode(code.Id)
	service = qservice.NewService(setting)

	// 启动微服务
	service.Run()
}
