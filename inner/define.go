package main

import (
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qconvert"
	"router/inner/blls"
	"router/inner/config"
	"router/inner/models"
)

const (
	Version   = "V1.0.0"   // 版本
	DefModule = "Route"    // 模块名称
	DefDesc   = "路由调度管理模块" // 模块描述
)

var (
	service *qservice.MicroService

	// 其他业务
	deviceCode string
	routeBll   *blls.Route
)

// 初始化
func onInit(moduleName string) {
	// 配置初始化
	config.Init(moduleName)

	// 业务初始化
	routeBll = blls.NewRouteBll(moduleName, deviceCode, service.SendRequest)

	// 如果没生成客户端唯一码，重新生成并重置客户端
	devCode, _ := qservice.DeviceCode.LoadFromFile()
	if devCode.IsEmpty() {

		code, err := routeBll.NewCode(service.IsRoot(), refs.newDeviceCode)
		if err != nil {
			panic(err)
		}
		devCode.Id = code
		// 保存到文件
		err = qservice.DeviceCode.SaveToFile(devCode)
		if err != nil {
			panic(err)
		}
		deviceCode = devCode.Id

		// 使用设备码重启连接
		service.ResetClient(devCode.Id)

		// 业务重新初始化
		routeBll.Stop()
		routeBll = nil
		routeBll = blls.NewRouteBll(moduleName, deviceCode, service.SendRequest)
	}

	// 输出设备码给启动器
	fmt.Println("[DeviceCode]:", deviceCode)
}

// 处理外部请求
func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	switch route {
	case "KnockDoor":
		return refs.knockDoor(qconvert.ToAny[map[string]string](ctx.Raw()))
	case "Request":
		info := qconvert.ToAny[models.RouteInfo](ctx.Raw())
		return routeBll.Req(info)
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
	case "ClientModuleList":
		routeBll.UploadClientModules(qconvert.ToAny[models.ServerInfo](ctx.Raw()))
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
