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
	devCode         string
	upAdapter       easyCon.IAdapter
	downRequestFunc qdefine.SendRequestHandler
}

func NewRouteBll(name, devCode string, downRequestFunc qdefine.SendRequestHandler) *Route {
	route := &Route{
		name:            name,
		devCode:         devCode,
		downRequestFunc: downRequestFunc,
	}

	// 如果配置了上级broker，则连接
	cfg := config.Config.UpMqtt
	if cfg.Addr != "" {
		setting := easyCon.NewSetting(fmt.Sprintf("%s.%s", name, devCode), cfg.Addr, route.onReq, route.onStatusChanged)
		setting.UID = cfg.UId
		setting.PWD = cfg.Pwd
		setting.TimeOut = time.Duration(cfg.TimeOut) * time.Second
		setting.ReTry = cfg.Retry
		setting.LogMode = easyCon.ELogMode(cfg.LogMode)
		route.upAdapter = easyCon.NewMqttAdapter(setting)
	}

	return route
}

func (r *Route) Stop() {
	if r.upAdapter != nil {
		r.upAdapter.Stop()
		r.upAdapter = nil
	}
}

func (r *Route) Req(info models.RouteInfo) (any, error) {
	if info.Module == "" {
		return nil, errors.New("moduleName is nil")
	}

	// 非路由请求
	if strings.Contains(info.Module, "/") == false {
		rs, err := r.upRequest(info.Module, info.Route, info.Content)
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

	// 如果是当前设备或者是根
	if sp[0] == r.devCode || r.devCode == "" {
		newModule := ""
		if r.devCode == "" {
			// 根不用去
			newModule = info.Module
		} else {
			// 去掉头部
			newModule = strings.Replace(info.Module, fmt.Sprintf("%s/", r.devCode), "", 1)
		}
		// 已经是最底层路由
		if strings.Contains(newModule, "/") == false {
			rs, err := r.upRequest(newModule, info.Route, info.Content)
			if err != nil {
				return nil, err
			}
			return rs, nil
		}
		// 未到底层，继续向下级路由请求
		newParams["Module"] = newModule
		// 截取下级设备码
		sp = strings.Split(newModule, "/")
		rs, err := r.downRequestFunc(fmt.Sprintf("%s.%s", r.name, sp[0]), "Request", newParams)
		if err != nil {
			return nil, err
		}
		return rs, nil
	} else {
		// 向上机路由请求
		rs, err := r.upRequest(r.name, "Request", newParams)
		if err != nil {
			return nil, err
		}
		return rs, nil
	}
}

func (r *Route) upRequest(module, route string, content any) (any, error) {
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

func (r *Route) UploadClientModules(info models.ServerInfo) {
	if r.devCode != info.DeviceCode || r.upAdapter == nil {
		// 底层路由不用上传
		return
	}

	device, _ := qservice.DeviceCode.LoadFromFile()

	up := map[string]any{}
	up["Id"] = info.DeviceCode
	up["Name"] = device.Name
	up["Modules"] = info.Modules
	r.upAdapter.Req("ClientManager", "KnockDoor", info)
}

func (r *Route) NewCode(isRoot bool, downNewCodeFunc func() (string, error)) (string, error) {
	if r.upAdapter != nil {
		param := map[string]any{
			"IsRoot": isRoot,
		}
		// 桥接模式，问上级服务请求客户端
		resp := r.upAdapter.Req("ClientManager", "NewDeviceCode", param)
		if resp.RespCode == easyCon.ERespSuccess {
			return resp.Content.(string), nil
		}
		if resp.Error != "" {
			return "", errors.New(resp.Error)
		}
		return "", errors.New(fmt.Sprintf("%d", resp.RespCode))
	}
	// 根级或者最底层，则直接问同级的服务请求
	return downNewCodeFunc()
}
