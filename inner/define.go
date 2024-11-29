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

	// 启动
	routeBll.Start()

	//onLog(qdefine.ELogError, "InstrDecodePlugin", errors.New("未将对象引用到实例化"))

	// 输出信息
	fmt.Printf("[DeviceKnock]:%s^%s\n", routeBll.GetDevId(), routeBll.GetDevName())
}

// 处理外部请求
func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	switch route {

	// 由下级路由模块请求，生成一个新的设备ID
	case "NewDeviceId":
		return deviceBll.NewDeviceId()

		// 由下级模块请求，获取服务器的设备ID
	case "ServerDevId":
		return routeBll.GetDevId(), nil

		// 由下级模块请求，敲门
	case "KnockDoor":
		info := qconvert.ToAny[models.DeviceKnock](ctx.Raw())
		return deviceBll.KnockDoor(info, routeBll.GetDevId())

		// 下级路由请求，发送心跳
	case "Heart":
		id := ctx.GetString("id")
		info := map[string]models.DeviceAlarm{}
		ctx.GetStruct("Info", &info)
		deviceBll.AddHeart(id, info)
		return true, nil

		// 下级路由请求，发送错误日志
	case "ErrorLog":
		devId := ctx.GetString("id")
		module := ctx.GetString("module")
		title := ctx.GetString("title")
		err := ctx.GetString("error")
		deviceBll.AddError(devId, module, title, err)
		return true, nil

		// 仅获取所有报警设备列表
	case "AlarmDeviceList":
		return deviceBll.GetDeviceAlarm()

		// 获取所有设备列表
	case "AllDeviceList":
		return deviceBll.GetDeviceList()

		// 获取指定设备包含的模块列表
	case "ModuleList":
		devices := qconvert.ToAny[[]string](ctx.Raw())
		return deviceBll.GetModuleList(devices)

		// 路由请求
	case "Request":
		info := qconvert.ToAny[models.RouteInfo](ctx.Raw())
		return routeBll.Req(info)

		// 反向ping测试
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
