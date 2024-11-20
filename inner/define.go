package main

import (
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qconvert"
	"router/inner/blls"
	"router/inner/daos"
	"router/inner/models"
)

const (
	Version   = "V1.0.0" // 版本
	DefModule = "Route"  // 模块名称
	DefDesc   = "路由模块"   // 模块描述
)

var (
	service *qservice.MicroService

	// 其他业务
	deviceBll *blls.Device
	routeBll  *blls.Route
)

// 初始化
func onInit(moduleName string) {
	// 数据库初始化
	daos.Init(moduleName)

	// 业务初始化
	routeBll = blls.NewRouteBll(moduleName)
	routeBll.DownRequestFunc = service.SendRequest
	routeBll.ResetClientFunc = service.ResetClient
	deviceBll = blls.NewDeviceBll()
	deviceBll.UpKnockDoorFunc = routeBll.KnockDoor

	// 启动
	routeBll.Start()

	// 输出信息
	fmt.Printf("[DeviceInfo]:%s^%s\n", routeBll.GetDevId(), routeBll.GetDevName())
}

// 处理外部请求
func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	switch route {
	case "NewDeviceId":
		return deviceBll.NewDeviceId(routeBll.GetDevId())
	case "KnockDoor":
		info := qconvert.ToAny[models.DeviceInfo](ctx.Raw())
		return deviceBll.KnockDoor(info)
	case "Request":
		info := qconvert.ToAny[models.RouteInfo](ctx.Raw())
		return routeBll.Req(info)
	case "Ping":
		fmt.Println("[Ping]:", "OK")
		return true, nil
	}
	return nil, errors.New("route Not Matched")
}

// 处理外部通知
func onNoticeHandler(route string, ctx qdefine.Context) {
	switch route {

	}
}

func onRetainNoticeHandler(route string, ctx qdefine.Context) {
	switch route {

	}
}

// 发送通知
func onNotice(route string, content any) {
	service.SendNotice(route, content)
}

// 发送日志
func onLog(logType qdefine.ELog, content string, err error) {
	service.SendLog(logType, content, err)
}
