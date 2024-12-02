package blls

import (
	"router/inner/config"
	"router/inner/daos"
	"router/inner/models"
	"sort"
	"strings"
	"sync"
)

type device struct {
	monitorBll   *monitor
	lock         *sync.Mutex
	upperDevice  models.DeviceKnock           // 上层路由设备信息
	localDevices map[string]models.DeviceInfo // 自己路由设备缓存列表
	alarmCaches  map[string]models.DeviceAlarm
}

func newDeviceBll() *device {
	d := &device{
		lock:         &sync.Mutex{},
		upperDevice:  models.DeviceKnock{},
		localDevices: map[string]models.DeviceInfo{},
		alarmCaches:  map[string]models.DeviceAlarm{},
	}
	d.monitorBll = newMonitorBll(d.onMonitorChanged, d.onHeartChanged)
	return d
}

func (d *device) Start() {
	// 生成本级设备信息
	dev := d.localDevices[config.DeviceId()]
	dev.Id = config.DeviceId()
	dev.Name = config.DeviceName()
	dev.Modules = models.ModuleCollection{}
	dev.FullUrl = strings.Trim(d.upperDevice.FullUrl+"/"+dev.Id, "/")
	sp := strings.Split(dev.FullUrl, "/")
	if len(sp) >= 2 {
		dev.Parent = sp[len(sp)-2]
	}
	d.localDevices[config.DeviceId()] = dev

	// 启动监控
	d.monitorBll.Start()
}

// AddHeart 添加下级路由发送的心跳和报警信息
func (d *device) AddHeart(devId string, routeHearts map[string]models.DeviceAlarm) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.monitorBll.AddHeart(devId)
	for k, v := range routeHearts {
		a := d.alarmCaches[k]
		a.Id = v.Id
		a.Name = v.Name
		a.Parent = v.Parent
		a.FullUrl = v.FullUrl
		a.Alarms = v.Alarms
		d.alarmCaches[k] = a
	}
}

func (d *device) GetUpperModules() map[string]string {
	d.lock.Lock()
	defer d.lock.Unlock()

	modules := map[string]string{}
	for _, m := range d.upperDevice.Modules {
		modules[m.Name] = d.upperDevice.Id
	}

	return modules
}

func (d *device) SetUpperDevice(info models.DeviceKnock) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.upperDevice = info
}

func (r *device) GetDeviceCache() (models.DeviceKnock, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.localDevices[config.DeviceId()].DeviceKnock, nil
}

func (d *device) SetLocalDevice(infos map[string]models.DeviceKnock) map[string]models.DeviceKnock {
	d.lock.Lock()
	defer d.lock.Unlock()

	// 添加到缓存
	for id, door := range infos {
		dev := d.localDevices[id]
		dev.Modules.Add(door.Modules)
		if door.FullUrl != "" {
			dev.Id = door.Id
			dev.Name = door.Name
			dev.FullUrl = door.FullUrl
			sp := strings.Split(dev.FullUrl, "/")
			if len(sp) >= 2 {
				dev.Parent = sp[len(sp)-2]
			}
		}
		d.localDevices[id] = dev
	}

	knocks := map[string]models.DeviceKnock{}
	for k, v := range d.localDevices {
		knocks[k] = v.DeviceKnock
	}

	// 写入到数据库
	if daos.DeviceDao == nil {
		return knocks
	}

	return knocks
}

func (d *device) GetAlarmCaches() map[string]models.DeviceAlarm {
	d.lock.Lock()
	defer d.lock.Unlock()

	return d.alarmCaches
}

func (d *device) onMonitorChanged(tp string, content any) {
	d.lock.Lock()
	defer d.lock.Unlock()

	localId := config.DeviceId()

	dev := d.localDevices[localId]
	alarm := d.alarmCaches[localId]
	alarm.Id = dev.Id
	alarm.Name = dev.Name
	alarm.Parent = dev.Parent
	alarm.FullUrl = dev.FullUrl

	switch tp {
	case "CPU":
		dev.Cpu = content.(models.CpuMemState)
		alarm.Set(tp, !dev.Cpu.IsOk, "alarm", dev)

	case "MEM":
		dev.Memory = content.(models.CpuMemState)
		alarm.Set(tp, !dev.Memory.IsOk, "alarm", dev)

	case "DISK":
		dev.Disk = content.([]models.DiskState)
		value := ""
		for _, d := range dev.Disk {
			if d.IsOk == false {
				value += d.Name + " "
			}
		}
		alarm.Set(tp, value != "", "alarm", dev)

	case "PROCESS":
		dev.Process = content.([]models.ProcessState)
		value := ""
		for _, p := range dev.Process {
			if p.IsOk == false {
				value += p.Name + " exit\n"
			}
		}
		alarm.Set(tp, value != "", strings.Trim(value, "\n"), dev)
	}
	d.localDevices[localId] = dev
	d.alarmCaches[localId] = alarm
}

func (d *device) onHeartChanged(ids map[string]bool) {
	d.lock.Lock()
	defer d.lock.Unlock()

	save := map[string]models.DeviceInfo{}
	for k, v := range d.localDevices {
		dev := d.localDevices[k]
		alarm := d.alarmCaches[k]

		v.IsOnline = true
		if _, ok := ids[k]; ok {
			v.IsOnline = false
		}
		alarm.Set("Network", !v.IsOnline, "offline", dev)

		d.alarmCaches[k] = alarm
		save[k] = v
	}
	for k, v := range save {
		d.localDevices[k] = v
	}
}

func (d *device) GetDeviceAlarm() (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	list := make([]models.DeviceAlarm, 0)
	for _, v := range d.alarmCaches {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Parent == list[j].Parent {
			return list[i].Id < list[j].Id
		}
		return list[i].Parent < list[j].Parent
	})
	return list, nil
}

func (d *device) GetDeviceList() (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	list := make([]map[string]any, 0)
	for _, v := range d.localDevices {
		list = append(list, map[string]any{
			"Id":       v.Id,
			"Name":     v.Name,
			"Parent":   v.Parent,
			"RouteUrl": v.FullUrl,
		})
	}
	return list, nil
}

func (d *device) GetDeviceDetail() (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	return d.localDevices[config.DeviceId()], nil
}
