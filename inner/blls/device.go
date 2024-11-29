package blls

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/kamioair/qf/qservice"
	"router/inner/daos"
	"router/inner/models"
	"strings"
	"sync"
	"time"
)

type Device struct {
	UpKnockDoorFunc func(info models.DeviceKnock)            // 向上级敲门方法
	UpSendHeartFunc func(info map[string]models.DeviceAlarm) // 向上级发送心跳

	monitorBll   *monitor
	lock         *sync.Mutex
	localDevId   string
	alarmCaches  map[string]models.DeviceAlarm
	deviceCaches map[string]models.DeviceInfo
	upKnockChan  chan models.DeviceKnock
}

// NewDeviceBll 构造
func NewDeviceBll() *Device {
	dev := &Device{
		lock:         &sync.Mutex{},
		alarmCaches:  make(map[string]models.DeviceAlarm),
		deviceCaches: make(map[string]models.DeviceInfo),
		upKnockChan:  make(chan models.DeviceKnock),
	}
	dev.monitorBll = newMonitorBll(dev.onMonitorChanged, dev.onHeartChanged)

	go dev.upLoop()
	return dev
}

// Start 启动
func (d *Device) Start(devId string) {
	d.localDevId = devId
	d.monitorBll.Start()
}

// NewDeviceId 给下级路由分配一个新的设备ID
func (d *Device) NewDeviceId() (any, error) {
	// 返回新的ID
	return uuid.NewString(), nil
}

// AddHeart 添加下级路由发送的心跳和报警信息
func (d *Device) AddHeart(devId string, routeHearts map[string]models.DeviceAlarm) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.monitorBll.AddHeart(devId)
	for k, v := range routeHearts {
		a := d.alarmCaches[k]
		a.Alarms = v.Alarms
		d.alarmCaches[k] = a
	}
}

// AddError 添加下级路由发送的错误信息
func (d *Device) AddError(devId string, module string, title string, err string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	alarm := d.alarmCaches[devId]
	alarm.Set(module, true, title)
	d.alarmCaches[devId] = alarm

	dev := d.deviceCaches[devId]
	for i := 0; i < len(dev.Modules); i++ {
		mod := dev.Modules[i]
		if mod.Name == module {
			exist := false
			for j := 0; j < len(mod.Errors); j++ {
				if mod.Errors[j].Name == title {
					mod.Errors[j].Value = err
					exist = true
					break
				}
			}
			if !exist {
				mod.Errors = append(mod.Errors, models.Item{
					Name:  title,
					Value: err,
				})
			}
			break
		}
		dev.Modules[i] = mod
	}
	d.deviceCaches[devId] = dev

	// 记录到日志文件
}

// KnockDoor 连接到Broker的所有模块敲门处理，用于记录所有设备和设备包含的模块信息到数据库中
func (d *Device) KnockDoor(info models.DeviceKnock, devId string) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	find := &daos.Device{}
	if info.Parent == "" || info.Id == devId {
		// 本级路由
		find, _ = daos.DeviceDao.GetCondition("code = ?", "local")
		find.Name = info.Id
	} else {
		// 下级路由
		find, _ = daos.DeviceDao.GetCondition("code = ?", info.Id)
		if find == nil {
			find = &daos.Device{Code: info.Id}
		}
		find.Name = info.Name
		find.Parent = info.Parent
	}
	finalModules := d.joinModules(find.Modules, info.Modules)
	js, _ := json.Marshal(finalModules)
	find.Modules = string(js)

	// 更新数据库
	err := daos.DeviceDao.Save(find)
	if err != nil {
		return false, err
	}

	// 添加到缓存
	local, _ := daos.DeviceDao.GetCondition("code = ?", "local")
	dev := d.deviceCaches[info.Id]
	dev.Id = info.Id
	dev.Name = info.Name
	dev.Parent = info.Parent
	dev.Modules = finalModules
	if local.Name == dev.Id {
		dev.RouteUrl = fmt.Sprintf("%s/%s", local.Parent, dev.Id)
	} else {
		dev.RouteUrl = fmt.Sprintf("%s/%s/%s", local.Parent, local.Name, dev.Id)
	}

	d.deviceCaches[info.Id] = dev

	// 如果有上级路由，则向上继续敲门
	d.upKnockChan <- info

	// 成功
	return true, nil
}

// GetDeviceAlarm 获取所有报警的设备列表
func (d *Device) GetDeviceAlarm() ([]models.DeviceAlarm, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	list := []models.DeviceAlarm{}
	for _, v := range d.alarmCaches {
		list = append(list, v)
	}
	return list, nil
}

func (d *Device) GetDeviceList() ([]models.DeviceInfo, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	list := []models.DeviceInfo{}
	for _, v := range d.deviceCaches {
		list = append(list, v)
	}
	return list, nil
}

func (d *Device) GetModuleList(devCodes []string) (map[string]string, error) {
	finals := map[string]string{}

	// 先查找服务器的所有模块
	for _, devCode := range devCodes {
		if devCode == "local" {
			local, err := daos.DeviceDao.GetCondition("code = ?", devCode)
			if err != nil {
				return finals, err
			}
			devInfo, _ := qservice.DeviceCode.LoadFromFile()
			modules := make([]models.ModuleInfo, 0)
			_ = json.Unmarshal([]byte(local.Modules), &modules)
			for _, m := range modules {
				finals[m.Name] = devInfo.Id
			}
		} else {
			// 再查找指定设备的模块
			device, err := daos.DeviceDao.GetCondition("code = ?", devCode)
			if err != nil {
				return finals, err
			}
			if device != nil {
				modules := make([]models.ModuleInfo, 0)
				_ = json.Unmarshal([]byte(device.Modules), &modules)
				for _, m := range modules {
					finals[m.Name] = device.Code
				}
			}
		}
	}

	return finals, nil
}

func (d *Device) onMonitorChanged(tp string, content any) {
	d.lock.Lock()
	defer d.lock.Unlock()

	dev := d.deviceCaches[d.localDevId]
	alarm := d.alarmCaches[d.localDevId]

	switch tp {
	case "CPU":
		dev.Cpu = content.(models.CpuMemState)
		alarm.Set(tp, !dev.Cpu.IsOk, "alarm")

	case "MEM":
		dev.Memory = content.(models.CpuMemState)
		alarm.Set(tp, !dev.Memory.IsOk, "alarm")

	case "DISK":
		dev.Disk = content.([]models.DiskState)
		value := ""
		for _, d := range dev.Disk {
			if d.IsOk == false {
				value += d.Name + " "
			}
		}
		alarm.Set(tp, value != "", "alarm")

	case "PROCESS":
		dev.Process = content.([]models.ProcessState)
		value := ""
		for _, p := range dev.Process {
			if p.IsOk == false {
				value += p.Name + "exit;"
			}
		}
		alarm.Set(tp, value != "", strings.Trim(value, ";"))
	}

	d.deviceCaches[d.localDevId] = dev
	d.alarmCaches[d.localDevId] = alarm
}

func (d *Device) onHeartChanged(ids map[string]bool) {
	d.lock.Lock()
	defer d.lock.Unlock()

	save := map[string]models.DeviceInfo{}
	for k, v := range d.deviceCaches {
		alarm := d.alarmCaches[k]

		v.IsOnline = true
		if _, ok := ids[k]; ok {
			v.IsOnline = false
		}
		alarm.Set("Network", !v.IsOnline, "offline")

		d.alarmCaches[k] = alarm
		save[k] = v
	}
	for k, v := range save {
		d.deviceCaches[k] = v
	}
}

func (d *Device) upLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case info := <-d.upKnockChan:
			newInfo := models.DeviceKnock{
				Id:      info.Id,
				Name:    info.Name,
				Parent:  info.Parent,
				Modules: info.Modules,
			}
			go d.UpKnockDoorFunc(newInfo)

		case <-ticker.C:
			// 向上级路由模块发送请求
			d.lock.Lock()
			alarm := map[string]models.DeviceAlarm{}
			for k, v := range d.alarmCaches {
				alarm[k] = v
			}
			d.lock.Unlock()

			go d.UpSendHeartFunc(alarm)
		}
	}
}

func (d *Device) joinModules(oldModulesStr string, newModules []models.ModuleInfo) []models.ModuleInfo {
	finalModules := make([]models.ModuleInfo, 0)
	if oldModulesStr != "" {
		_ = json.Unmarshal([]byte(oldModulesStr), &finalModules)
	}
	// 遍历
	for _, nm := range newModules {
		exist := false
		for _, om := range finalModules {
			if om.Name == nm.Name {
				om.Desc = nm.Desc
				om.Version = nm.Version
				exist = true
			}
		}
		if exist == false {
			finalModules = append(finalModules, nm)
		}
	}
	return finalModules
}
