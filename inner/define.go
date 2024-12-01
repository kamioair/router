package main

import (
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qconvert"
	"router/inner/blls"
	"router/inner/config"
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
	if config.Config.Mode == config.ERouteServer {
		// 添加设备登记数据库
		daos.Init(moduleName)
	}

	// 业务初始化
	routeBll = blls.NewRouteBll(moduleName)
	routeBll.DownRequestFunc = service.SendRequest
	routeBll.ResetClientFunc = service.ResetClient
	deviceBll = blls.NewDeviceBll()
	deviceBll.UpKnockDoorFunc = routeBll.KnockDoor
	deviceBll.UpSendHeartFunc = routeBll.SendHeart
	deviceBll.SendRequest = service.SendRequest

	// 启动
	routeBll.Start()
	deviceBll.Start(routeBll.GetDevId(), routeBll.GetParentDev())

	// 输出信息
	fmt.Printf("[DeviceKnock]:%s^%s\n", routeBll.GetDevId(), routeBll.GetDevName())
}

// 处理外部请求
func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	switch route {
	//-------------------------------------------
	//  以下是服务级路由模块，提供给所有功能模块使用的方法

	case "Request": // 跨路由请求
		model := qconvert.ToAny[models.RouteInfo](ctx.Raw())
		return routeBll.Req(model)
	case "NewDeviceId": // 下级路由请求生成一个新的设备ID
		return deviceBll.NewDeviceId()
	case "DiscoveryList": // 返回服务id和相关模块登记列表，用于客户端进行模块发现
		devices := qconvert.ToAny[[]string](ctx.Raw())
		return deviceBll.GetDiscoveryList(devices)

	//-------------------------------------------
	//  以下方法执行方有两类
	//   1、同级模块给同级路由模块发送（客户端级的模块给同客户端级的路由模块发送，服务级模块给服务级路由发送）
	//   2、服务级路由继续给上级路由模块发送

	case "KnockDoor": // 模块敲门
		info := qconvert.ToAny[models.DeviceKnock](ctx.Raw())
		return deviceBll.KnockDoor(info)
	case "Heart": // 模块心跳
		id := ctx.GetString("id")
		info := map[string]models.DeviceAlarm{}
		ctx.GetStruct("Info", &info)
		deviceBll.AddHeart(id, info)
		return true, nil
	//case "ErrorLog": // 模块错误日志
	//	devId := ctx.GetString("id")
	//	module := ctx.GetString("module")
	//	title := ctx.GetString("title")
	//	err := ctx.GetString("error")
	//	deviceBll.AddError(devId, module, title, err)
	//	return true, nil

	//-------------------------------------------
	//  以下方法提供给前端管理页面访问
	case "AlarmDeviceList": // 仅获取所有报警设备列表
		return deviceBll.GetDeviceAlarm()
	case "AllDeviceList": // 获取所有设备列表
		return deviceBll.GetDeviceList()
	case "GetDeviceDetail": // 获取当前设备的详细信息
		return deviceBll.GetDeviceDetail()
	case "Ping": // 反向ping测试
		return fmt.Println("[Ping]:", "OK")
	}
	return nil, errors.New("route Not Matched")
}

func onLoadServDiscoveryList() string {
	return routeBll.GetDiscoveryList()
}

// 处理外部通知
func onNoticeHandler(route string, ctx qdefine.Context) {
	switch route {

	}
}

func onStatusHandler(route string, ctx qdefine.Context) {
	//if route == "StatusInputRetain" {
	//	ctx.
	//}
}

// 发送通知
func onNotice(route string, content any) {
	service.SendNotice(route, content)
}

// 发送日志
func onLog(logType qdefine.ELog, content string, err error) {
	service.SendLog(logType, content, err)
}
