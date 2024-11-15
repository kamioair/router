package main

import (
	"github.com/google/uuid"
	"github.com/kamioair/qf/qservice"
	"router/inner/blls"
)

func main() {
	// 防止冲突，如果没有请求到客户端ID时，先随机生成一个
	code, err := blls.DeviceCode.LoadFromFile()
	if err != nil {
		id, _ := uuid.NewUUID()
		code = id.String()
	}
	setting := qservice.NewSetting(DefModule, DefDesc, Version).
		BindInitFunc(onInit).
		BindReqFunc(onReqHandler).
		BindNoticeFunc(onNoticeHandler).
		SetDeviceCode(code)
	service = qservice.NewService(setting)

	// 启动微服务
	service.Run()
}
