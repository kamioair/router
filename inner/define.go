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
	"time"
)

const (
	Version   = "V1.0.0" // 版本
	DefModule = "Route"  // 模块名称
	DefDesc   = "路由模块"   // 模块描述
)

var (
	service *qservice.MicroService

	// 其他业务
	routeBll   *blls.Route
	initFinish bool
)

// 初始化
func onInit(moduleName string) {
	// 数据库初始化
	if service.Setting().Mode == qservice.EModeServer {
		// 添加设备登记数据库
		daos.Init(moduleName)
	}

	// 业务初始化
	routeBll = blls.NewRouteBll(service.Adapter(), onNotice)
	routeBll.Start()

	// 输出信息
	fmt.Printf("[DeviceKnock]:%s^%s\n", config.DeviceId(), config.DeviceName())
	initFinish = true
}

// 处理外部请求
func onReqHandler(route string, ctx qdefine.Context) (any, error) {
	switch route {

	//-------------------------------------------
	//  以下由基于Qf框架的模块发送请求
	case "KnockDoor": // 模块敲门
		doors := qconvert.ToAny[map[string]models.DeviceKnock](ctx.Raw())
		return routeBll.KnockDoor(doors)
	case "Request": // 跨路由请求
		model := qconvert.ToAny[models.RouteInfo](ctx.Raw())
		return routeBll.Request(model)
	case "CustomAlarm": // 模块的自定义警报
		alarmType := ctx.GetString("type")
		alarmValue := ctx.GetString("value")
		return routeBll.AddAlarm(alarmType, alarmValue)

	//-------------------------------------------
	//  以下仅由路由模块向上层路由模块发送请求
	//    客户端路由 -》 服务端的根路由
	//    服务端路由 -》 上级服务端的根路由
	case "GetDeviceCache": // 请求服务器设备信息
		return routeBll.GetDeviceCache()
	case "NewDeviceId": // 申请一个新的Id
		return routeBll.NewDeviceId()
	case "Heart": // 发送心跳
		alarm := qconvert.ToAny[struct {
			Id   string
			Info map[string]models.DeviceAlarm
		}](ctx.Raw())
		routeBll.AddHeart(alarm.Id, alarm.Info)
		return true, nil

	//-------------------------------------------
	//  以下由前端管理页面发送请求
	case "AlarmDeviceList": // 仅获取所有报警设备列表
		return routeBll.GetDeviceAlarm()
	case "AllDeviceList": // 获取所有设备列表
		return routeBll.GetDeviceList()
	case "GetDeviceDetail": // 获取当前设备的详细信息
		return routeBll.GetDeviceDetail()
	case "Ping": // 反向ping测试
		return fmt.Println("[Ping]:", "OK")
	}
	return nil, errors.New("route Not Matched")
}

// 处理外部通知
func onNoticeHandler(route string, ctx qdefine.Context) {
	switch route {

	}
}

func onStatusHandler(route string, ctx qdefine.Context) {
	switch route {

	}
}

func onCommStateHandler(state qdefine.ECommState) {
	if state == qdefine.ECommStateLinked {
		if initFinish {
			go func() {
				time.Sleep(time.Second * 5)
				routeBll.ReKnockDoor()
			}()
		}
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
