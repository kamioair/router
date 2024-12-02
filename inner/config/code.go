package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qconfig"
	"github.com/kamioair/qf/utils/qio"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"runtime"
	"time"
)

func DeviceId() string {
	return device.info.Id
}

func DeviceName() string {
	return device.info.Name
}

var device deviceCode

type deviceCode struct {
	info qdefine.DeviceInfo
}

// LoadFromFile 从文件中获取设备码
func (d *deviceCode) loadFromFile(mode qservice.EServerMode) {
	file := d.getCodeFile()
	if qio.PathExists(file) {
		// 文件存在，则从文件中获取设备信息
		str, err := qio.ReadAllString(file)
		if err != nil {
			goto newId
		}
		info := qdefine.DeviceInfo{}
		err = json.Unmarshal([]byte(str), &info)
		if err != nil {
			goto newId
		}
		d.info = info
		return
	}
	// 否则向上级路由请求一个新的ID
newId:
	var info qdefine.DeviceInfo
	if mode == qservice.EModeServer {
		// 说明是最顶级路由，直接分配一个固定的设备
		if UpMqtt.Addr == "" {
			info = qdefine.DeviceInfo{
				Id:   "root",
				Name: "Root Server",
			}
		} else {
			// 创建临时连接，并问上级路由模块请求
			setting := easyCon.NewSetting(fmt.Sprintf("Route.%s", qdefine.NewUUID()+".[TEMP]"), UpMqtt.Addr, onReq, onStatus)
			setting.UID = UpMqtt.UId
			setting.PWD = UpMqtt.Pwd
			setting.TimeOut = time.Duration(UpMqtt.TimeOut) * time.Second
			setting.ReTry = UpMqtt.Retry
			setting.LogMode = easyCon.ELogMode(UpMqtt.LogMode)
			adapter := easyCon.NewMqttAdapter(setting)
			time.Sleep(time.Second)
			resp := adapter.Req("Route", "NewDeviceId", nil)
			if resp.RespCode == easyCon.ERespSuccess {
				info = qdefine.DeviceInfo{Id: resp.Content.(string)}
			} else {
				panic(resp.Error)
			}
			// 执行请求
			adapter.Stop()
		}
	} else {
		// 客户端，直接问服务器的根路由请求
		broker := qdefine.BrokerConfig{
			Addr:    qconfig.Get("", "mqtt.addr", "ws://127.0.0.1:5002/ws"),
			UId:     qconfig.Get("", "mqtt.uid", ""),
			Pwd:     qconfig.Get("", "mqtt.pwd", ""),
			LogMode: qconfig.Get("", "mqtt.logMode", "NONE"),
			TimeOut: qconfig.Get("", "mqtt.timeOut", 3000),
			Retry:   qconfig.Get("", "mqtt.retry", 3),
		}
		setting := easyCon.NewSetting(fmt.Sprintf("Route.%s", qdefine.NewUUID()+".[TEMP]"), broker.Addr, onReq, onStatus)
		setting.UID = broker.UId
		setting.PWD = broker.Pwd
		setting.TimeOut = time.Duration(broker.TimeOut) * time.Second
		setting.ReTry = broker.Retry
		setting.LogMode = easyCon.ELogMode(broker.LogMode)
		adapter := easyCon.NewMqttAdapter(setting)
		time.Sleep(time.Second)
		resp := adapter.Req("Route", "NewDeviceId", nil)
		if resp.RespCode == easyCon.ERespSuccess {
			info = qdefine.DeviceInfo{Id: resp.Content.(string)}
		} else {
			panic(resp.Error)
		}
		// 执行请求
		adapter.Stop()
	}
	// 保存文件
	err := d.saveToFile(info)
	if err != nil {
		panic(err)
	}
	d.info = info
}

func (d *deviceCode) saveToFile(info qdefine.DeviceInfo) error {
	// 写入文件
	file := d.getCodeFile()
	if file == "" {
		return errors.New("deviceCode file not find")
	}
	str, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	err = qio.WriteAllBytes(file, str, false)
	if err != nil {
		return err
	}
	return nil
}

func (d *deviceCode) getCodeFile() string {
	root := qio.GetCurrentRoot()
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s\\Program Files\\Qf\\device", root)
	case "linux":
		return "/usr/qf/device"
	}
	return ""
}

func onStatus(adapter easyCon.IAdapter, status easyCon.EStatus) {

}

func onReq(pack easyCon.PackReq) (easyCon.EResp, any) {
	return easyCon.ERespRouteNotFind, nil
}
