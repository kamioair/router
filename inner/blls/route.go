package blls

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"router/inner/config"
	"router/inner/models"
	"strings"
	"time"
)

type Route struct {
	name          string
	devCode       string
	downAdapter   easyCon.IAdapter
	upRequestFunc qdefine.SendRequestHandler
}

func NewRouteBll(name, devCode string, upRequestFunc qdefine.SendRequestHandler) *Route {
	route := &Route{
		name:          name,
		devCode:       devCode,
		upRequestFunc: upRequestFunc,
	}

	// 如果配置了下级broker，则连接
	if config.Config.DownMqtt.Addr != "" {
		cfg := config.Config.DownMqtt
		setting := easyCon.NewSetting(fmt.Sprintf("%s.%s", name, devCode), cfg.Addr, route.onReq, route.onStatusChanged)
		setting.UID = cfg.UId
		setting.PWD = cfg.Pwd
		setting.TimeOut = time.Duration(cfg.TimeOut) * time.Second
		setting.ReTry = cfg.Retry
		setting.LogMode = easyCon.ELogMode(cfg.LogMode)
		route.downAdapter = easyCon.NewMqttAdapter(setting)
	}

	return route
}

func (r *Route) Req(info models.RouteInfo) (any, error) {
	if info.Module == "" {
		return nil, errors.New("moduleName is nil")
	}

	// 非路由请求
	if strings.Contains(info.Module, "/") == false {
		rs, err := r.downRequest(info.Module, info.Route, info.Content)
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
			rs, err := r.downRequest(newModule, info.Route, info.Content)
			if err != nil {
				return nil, err
			}
			return rs, nil
		}
		// 未到底层，继续向下级路由请求
		newParams["Module"] = newModule
		// 截取下级设备码
		sp = strings.Split(newModule, "/")
		rs, err := r.downRequest(fmt.Sprintf("%s.%s", r.name, sp[0]), "Request", newParams)
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

func (r *Route) downRequest(module, route string, content any) (any, error) {
	if r.downAdapter != nil {
		resp := r.downAdapter.Req(module, route, content)
		if resp.RespCode == easyCon.ERespSuccess {
			return resp.Content, nil
		}
		if resp.Error != "" {
			return nil, errors.New(resp.Error)
		}
		return nil, errors.New(fmt.Sprintf("%d", resp.RespCode))
	}
	return r.upRequestFunc(module, route, content)
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
