package blls

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/kamioair/qf/utils/qconvert"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"router/inner/config"
	"router/inner/models"
	"strings"
	"sync"
	"time"
)

type Route struct {
	upperAdapter easyCon.IAdapter // 上层Broker访问器
	localAdapter easyCon.IAdapter // 自己Broker访问器
	lock         *sync.Mutex
	deviceBll    *device
}

func NewRouteBll(localAdapter easyCon.IAdapter) *Route {
	r := &Route{
		localAdapter: localAdapter,
	}
	// 如果有上层配置，则连接
	if config.UpMqtt.Addr != "" {
		setting := easyCon.NewSetting(fmt.Sprintf("Route.%s", config.DeviceId()), config.UpMqtt.Addr, r.onReq, r.onStatus)
		setting.UID = config.UpMqtt.UId
		setting.PWD = config.UpMqtt.Pwd
		setting.TimeOut = time.Duration(config.UpMqtt.TimeOut) * time.Second
		setting.ReTry = config.UpMqtt.Retry
		setting.LogMode = easyCon.ELogMode(config.UpMqtt.LogMode)
		r.upperAdapter = easyCon.NewMqttAdapter(setting)
		time.Sleep(time.Second)
	}
	// 其他初始化
	r.deviceBll = newDeviceBll()
	return r
}

// Start 启动
func (r *Route) Start() {
	if r.upperAdapter != nil {
		// 服务路由，问上层路由要
		resp := r.upperAdapter.Req("Route", "GetDeviceCache", nil)
		if resp.RespCode == easyCon.ERespSuccess {
			r.deviceBll.SetUpperDevice(qconvert.ToAny[models.DeviceKnock](resp.Content))
		}
	} else if config.Mode.IsClient() {
		// 客户路由，问根路由要
		resp := r.localAdapter.Req("Route", "GetDeviceCache", nil)
		if resp.RespCode == easyCon.ERespSuccess {
			r.deviceBll.SetUpperDevice(qconvert.ToAny[models.DeviceKnock](resp.Content))
		}
	}
	// 启动设备
	r.deviceBll.Start()
	// 启动心跳
	go r.heartLoop()
}

// KnockDoor 敲门处理
func (r *Route) KnockDoor(doors map[string]models.DeviceKnock) (map[string]string, error) {
	// 添加到缓存
	list := r.deviceBll.SetLocalDevice(doors)

	// 客户端路由向服务器根路由敲门
	if config.Mode.IsClient() {
		r.localAdapter.Req("Route", "KnockDoor", list)
		// 返回上级的模块列表
		return r.deviceBll.GetUpperModules(), nil
	}

	// 服务路由且配置了上级Broker，向上级路由敲门
	if r.upperAdapter != nil {
		go r.upperAdapter.Req("Route", "KnockDoor", list)
	}
	return map[string]string{}, nil
}

// ReKnockDoor 重新敲门
func (r *Route) ReKnockDoor() {
	list := r.deviceBll.GetAllDeviceCache()

	// 客户端路由向服务器根路由敲门
	if config.Mode.IsClient() {
		r.localAdapter.Req("Route", "KnockDoor", list)
	}

	// 服务路由且配置了上级Broker，向上级路由敲门
	if r.upperAdapter != nil {
		go r.upperAdapter.Req("Route", "KnockDoor", list)
	}
}

// NewDeviceId 给下级路由分配一个新的设备ID
func (d *Route) NewDeviceId() (any, error) {
	// 返回新的ID
	return uuid.NewString(), nil
}

// AddAlarm 写入警报
func (r *Route) AddAlarm(alarmType string, value string) (any, error) {
	r.deviceBll.SetAlarm(alarmType, value)
	return true, nil
}

// GetDeviceCache 获取当前设备的信息
func (r *Route) GetDeviceCache() (models.DeviceKnock, error) {
	return r.deviceBll.GetLocalDeviceCache()
}

// Request 执行路由请求
func (r *Route) Request(info models.RouteInfo) (any, error) {
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

func (r *Route) AddHeart(id string, info map[string]models.DeviceAlarm) {
	r.deviceBll.AddHeart(id, info)
}

func (r *Route) GetDeviceAlarm() (any, error) {
	return r.deviceBll.GetDeviceAlarm()
}

func (r *Route) GetDeviceList() (any, error) {
	return r.deviceBll.GetDeviceList()
}

func (r *Route) GetDeviceDetail() (any, error) {
	return r.deviceBll.GetDeviceDetail()
}

func (r *Route) onReq(pack easyCon.PackReq) (easyCon.EResp, any) {
	switch pack.Route {
	case "Request":
		info := models.RouteInfo{}
		js, _ := json.Marshal(pack.Content)
		_ = json.Unmarshal(js, &info)
		rs, err := r.Request(info)
		if err != nil {
			return easyCon.ERespError, err.Error()
		}
		return easyCon.ERespSuccess, rs
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (r *Route) upRequestFunc(module, route string, content any) (any, error) {
	if r.upperAdapter != nil {
		resp := r.upperAdapter.Req(module, route, content)
		if resp.RespCode == easyCon.ERespSuccess {
			return resp.Content, nil
		}
		if resp.Error != "" {
			return nil, errors.New(resp.Error)
		}
		return nil, errors.New(fmt.Sprintf("%d", resp.RespCode))
	}
	resp := r.localAdapter.Req(module, route, content)
	if resp.RespCode == easyCon.ERespSuccess {
		return resp.Content, nil
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return nil, errors.New(fmt.Sprintf("%d", resp.RespCode))
}

func (r *Route) routeRequest(info models.RouteInfo) (any, error) {
	newParams := map[string]any{}
	newParams["Module"] = info.Module
	newParams["Route"] = info.Route
	newParams["Content"] = info.Content

	// 拆分路由
	sp := strings.Split(info.Module, "/")

	devCode := config.DeviceId()

	// 如果是当前设备或者是根
	if sp[0] == devCode || devCode == "root" || devCode == "" {
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
			// 如果是客户端，则补上ID，反之去掉
			if newModule == "Route" {
				if config.Mode.IsClient() {
					newModule = fmt.Sprintf("%s.%s", newModule, devCode)
				}
			} else {
				newModule = fmt.Sprintf("%s.%s", newModule, devCode)
			}
			resp := r.localAdapter.Req(newModule, info.Route, info.Content)
			if resp.RespCode == easyCon.ERespSuccess {
				return resp.Content, nil
			}
			if resp.Error != "" {
				return nil, errors.New(resp.Error)
			}
			return nil, errors.New(fmt.Sprintf("%d", resp.RespCode))
		}
		// 未到底层，继续向下级路由请求
		newParams["Module"] = newModule
		// 截取下级设备码
		sp = strings.Split(newModule, "/")
		resp := r.localAdapter.Req(fmt.Sprintf("Route.%s", sp[0]), "Request", newParams)
		if resp.RespCode == easyCon.ERespSuccess {
			return resp.Content, nil
		}
		if resp.Error != "" {
			return nil, errors.New(resp.Error)
		}
		return nil, errors.New(fmt.Sprintf("%d", resp.RespCode))
	} else {
		// 向上机路由请求
		rs, err := r.upRequestFunc("Route", "Request", newParams)
		if err != nil {
			return nil, err
		}
		return rs, nil
	}
}

func (r *Route) heartLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 向上级路由模块发送请求
			alarms := map[string]any{
				"Id":   config.DeviceId(),
				"Info": r.deviceBll.GetAlarmCaches(),
			}
			if config.Mode.IsClient() {
				go r.localAdapter.Req("Route", "Heart", alarms)
			} else {
				if r.upperAdapter != nil {
					go r.upperAdapter.Req("Route", "Heart", alarms)
				}
			}
		}
	}
}

func (r *Route) onStatus(adapter easyCon.IAdapter, status easyCon.EStatus) {

}
