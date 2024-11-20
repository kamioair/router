package blls

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"router/inner/config"
	"router/inner/models"
	"strings"
	"time"
)

type Route struct {
	name            string
	DeviceInfo      qdefine.DeviceInfo
	upAdapter       easyCon.IAdapter
	downRequestFunc qdefine.SendRequestHandler
	resetClientFunc func(devCode string)
}

func NewRouteBll(name string, downRequestFunc qdefine.SendRequestHandler, resetClientFunc func(devCode string)) *Route {
	route := &Route{
		name:            name,
		downRequestFunc: downRequestFunc,
		resetClientFunc: resetClientFunc,
	}

	// 获取本机设备信息
	route.getDeviceInfo()

	// 如果配置了上级路由，则连接
	route.initUpAdapter(route.DeviceInfo.Id)

	return route
}

func (r *Route) KnockDoor(info models.DeviceInfo) {
	if r.upAdapter == nil {
		return
	}
	_ = r.upAdapter.Req(r.name, "KnockDoor", info)
}

func (r *Route) getDeviceInfo() {
	// 先从本地文件获取
	device, err := qservice.DeviceCode.LoadFromFile()
	if err == nil {
		r.DeviceInfo = device
		return
	}

	// 本地没有，则上上级路由请求
	switch strings.ToLower(config.Config.Mode) {

	case "server": // 服务器模式
		// 创建临时连接，并问上级路由模块请求
		r.initUpAdapter(qdefine.NewUUID())
		ctx, err := r.upRequestFunc(r.name, "NewDeviceId", nil)
		if err != nil {
			panic(err)
		}
		device = qdefine.DeviceInfo{
			Id: ctx.(string),
		}
		// 得到客户端ID后，关闭临时连接
		r.upAdapter.Stop()
		r.upAdapter = nil

	case "root": // 根级模式
		// 固定ID
		device = qdefine.DeviceInfo{
			Id:   "root",
			Name: "Root Server",
		}

	default:
		// 客户端模式，向服务端路由请求
		ctx, err := r.downRequestFunc(r.name, "NewDeviceId", nil)
		if err != nil {
			panic(err)
		}
		device = qdefine.DeviceInfo{
			Id: ctx.Raw().(string),
		}
		// 使用新的客户端ID重启模块
		r.resetClientFunc(device.Id)
	}

	// 保存文件
	err = qservice.DeviceCode.SaveToFile(device)
	if err != nil {
		panic(err)
	}

	r.DeviceInfo = device
}

func (r *Route) initUpAdapter(devCode string) {
	cfg := config.Config.UpMqtt
	if cfg.Addr != "" {
		setting := easyCon.NewSetting(fmt.Sprintf("Route.%s", devCode), cfg.Addr, r.onReq, r.onStatusChanged)
		setting.UID = cfg.UId
		setting.PWD = cfg.Pwd
		setting.TimeOut = time.Duration(cfg.TimeOut) * time.Second
		setting.ReTry = cfg.Retry
		setting.LogMode = easyCon.ELogMode(cfg.LogMode)
		r.upAdapter = easyCon.NewMqttAdapter(setting)
	}
}

func (r *Route) onReq(pack easyCon.PackReq) (easyCon.EResp, any) {
	switch pack.Route {
	case "Request":
		info := models.RouteInfo{}
		js, _ := json.Marshal(pack.Content)
		_ = json.Unmarshal(js, &info)
		rs, err := r.Req(info)
		if err != nil {
			return easyCon.ERespError, err.Error()
		}
		return easyCon.ERespSuccess, rs
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (r *Route) onStatusChanged(adapter easyCon.IAdapter, status easyCon.EStatus) {

}

func (r *Route) upRequestFunc(module, route string, content any) (any, error) {
	if r.upAdapter != nil {
		resp := r.upAdapter.Req(module, route, content)
		if resp.RespCode == easyCon.ERespSuccess {
			return resp.Content, nil
		}
		if resp.Error != "" {
			return nil, errors.New(resp.Error)
		}
		return nil, errors.New(fmt.Sprintf("%d", resp.RespCode))
	}
	return r.downRequestFunc(module, route, content)
}

func (r *Route) Req(info models.RouteInfo) (any, error) {
	if info.Module == "" {
		return nil, errors.New("moduleName is nil")
	}

	// 非路由请求
	if strings.Contains(info.Module, "/") == false {
		rs, err := r.upRequestFunc(info.Module, info.Route, info.Content)
		if err != nil {
			return nil, err
		}
		return rs, nil
	}

	// 路由请求
	return r.routeRequest(info)
}

func (r *Route) routeRequest(info models.RouteInfo) (any, error) {
	newParams := map[string]any{}
	newParams["Module"] = info.Module
	newParams["Route"] = info.Route
	newParams["Content"] = info.Content

	// 拆分路由
	sp := strings.Split(info.Module, "/")

	devCode := r.DeviceInfo.Id

	// 如果是当前设备或者是根
	if sp[0] == devCode || devCode == "" {
		newModule := ""
		if devCode == "" {
			// 根不用去
			newModule = info.Module
		} else {
			// 去掉头部
			newModule = strings.Replace(info.Module, fmt.Sprintf("%s/", devCode), "", 1)
		}
		// 已经是最底层路由
		if strings.Contains(newModule, "/") == false {
			rs, err := r.upRequestFunc(newModule, info.Route, info.Content)
			if err != nil {
				return nil, err
			}
			return rs, nil
		}
		// 未到底层，继续向下级路由请求
		newParams["Module"] = newModule
		// 截取下级设备码
		sp = strings.Split(newModule, "/")
		rs, err := r.downRequestFunc(fmt.Sprintf("Route.%s", sp[0]), "Request", newParams)
		if err != nil {
			return nil, err
		}
		return rs, nil
	} else {
		// 向上机路由请求
		rs, err := r.upRequestFunc(r.name, "Request", newParams)
		if err != nil {
			return nil, err
		}
		return rs, nil
	}
}
