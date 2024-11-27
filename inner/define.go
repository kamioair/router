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
	deviceBll  *blls.Device
	routeBll   *blls.Route
	monitorBll *blls.Monitor
	alarmBll   *blls.Alarm
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
	monitorBll = blls.NewMonitorBll()
	alarmBll = blls.NewAlarmBll()
	alarmBll.SendDeviceState = routeBll.SendDeviceState
	alarmBll.OnNotice = onNotice
	deviceBll.GetAlarmsFunc = alarmBll.GetAlarms

	// 启动
	routeBll.Start()
	monitorBll.Start()
	alarmBll.Start()

	// 输出信息
	fmt.Printf("[DeviceInfo]:%s^%s\n", routeBll.GetDevId(), routeBll.GetDevName())
}

// 处理外部请求
func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	switch route {

	case "NewDeviceId": // 由下级路由模块请求，生成一个新的设备ID
		return deviceBll.NewDeviceId()

	case "ServerDevId": // 由下级模块请求，获取服务器的设备ID
		return routeBll.GetDevId(), nil

	case "KnockDoor": // 由下级模块请求，敲门
		info := qconvert.ToAny[models.DeviceInfo](ctx.Raw())
		return deviceBll.KnockDoor(info, routeBll.GetDevId())

	case "Heart":
		alarmBll.AddHeart(ctx.Raw().(string))
		return true, nil

	case "ErrorLog":
		devId := ctx.GetString("id")
		title := ctx.GetString("title")
		err := fmt.Sprintf("%s: %s", ctx.GetString("time"), ctx.GetString("error"))
		alarmBll.AddError(devId, title, err)
		return true, nil

	case "UploadDeviceState":
		alarmBll.AddDeviceState(ctx.Raw())
		return true, nil

	case "ModuleList":
		devices := qconvert.ToAny[[]string](ctx.Raw())
		return deviceBll.GetModuleList(devices)
	case "DeviceList":
		return deviceBll.GetDeviceList()
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
