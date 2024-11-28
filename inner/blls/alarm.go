package blls

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/qf/qservice"
	"github.com/kamioair/qf/utils/qconvert"
	"router/inner/config"
	"router/inner/models"
	"strings"
	"sync"
	"time"
)

type Alarm struct {
	lock        *sync.RWMutex         // 锁
	lastAlarm   string                // 上次上报的内容，用于比较是否需要上传
	deviceState map[string]deviceInfo // 设备的完整状态
	uploadState map[string]uploadInfo // 上传状态内容
	uploadChan  chan any
	devId       string

	SendDeviceState func(content any)
	OnNotice        func(route string, content any)
}

type deviceInfo struct {
	Heart   time.Time
	Cpu     bool
	Memory  bool
	Disk    map[string]bool
	Process map[string]bool
	Modules map[string]moduleInfo
}

type moduleInfo struct {
	Heart  time.Time
	Errors map[string]string
}

type uploadInfo struct {
	Alarms map[string]string
}

func NewAlarmBll() *Alarm {
	a := &Alarm{
		lock:        &sync.RWMutex{},
		deviceState: map[string]deviceInfo{},
		uploadState: map[string]uploadInfo{},
		uploadChan:  make(chan any),
		lastAlarm:   "{}",
	}
	return a
}

func (a *Alarm) Start(devId string) {
	a.devId = devId
	go a.checkLoop()
	go a.uploadLoop()
}

func (a *Alarm) GetAlarms() map[string]map[string]string {
	a.lock.Lock()
	defer a.lock.Unlock()

	finals := map[string]map[string]string{}
	for k, v := range a.uploadState {
		finals[k] = v.Alarms
	}
	return finals
}

func (a *Alarm) AddHeart(key string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	sp := strings.Split(key, "^")
	if len(sp) < 2 {
		return
	}

	device := a.deviceState[sp[0]]
	if device.Modules == nil {
		device.Modules = map[string]moduleInfo{}
	}

	// 更新模块心跳时间
	module := device.Modules[sp[1]]
	module.Heart = time.Now().Local()
	device.Modules[sp[1]] = module

	a.deviceState[sp[0]] = device
}

func (a *Alarm) AddError(key string, title, err string) {
	sp := strings.Split(key, "^")
	if len(sp) < 2 {
		return
	}
	device := a.deviceState[sp[0]]
	if device.Modules == nil {
		device.Modules = map[string]moduleInfo{}
	}

	module := device.Modules[sp[1]]
	if module.Errors == nil {
		module.Errors = map[string]string{}
	}
	module.Errors[title] = err

	device.Modules[sp[1]] = module
	a.deviceState[sp[0]] = device
}

func (a *Alarm) AddAlarmCpu(alarm bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	device := a.deviceState[a.devId]
	device.Cpu = alarm
	a.deviceState[a.devId] = device
}

func (a *Alarm) AddAlarmMemory(alarm bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	device := a.deviceState[a.devId]
	device.Memory = alarm
	a.deviceState[a.devId] = device
}

func (a *Alarm) AddAlarmDisk(alarm map[string]bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	device := a.deviceState[a.devId]
	device.Disk = map[string]bool{}
	for k, v := range alarm {
		device.Disk[k] = v
	}
	a.deviceState[a.devId] = device
}

func (a *Alarm) AddAlarmProcess(actives map[string]bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	device := a.deviceState[a.devId]
	device.Process = map[string]bool{}
	for k, v := range actives {
		device.Process[k] = v
	}
	a.deviceState[a.devId] = device
}

func (a *Alarm) AddDeviceState(raw any) {
	a.lock.Lock()
	defer a.lock.Unlock()

	content := qconvert.ToAny[map[string]uploadInfo](raw)
	for k, v := range content {
		a.uploadState[k] = v
	}

	// 通知前端有故障
	a.OnNotice("RouteDeviceAlarm", a.uploadState)
}

func (d *Alarm) GetDeviceStateDetail() (*models.DeviceStateFull, error) {
	full := &models.DeviceStateFull{
		Network: "online",
		Cpu:     "ok",
		Memory:  "ok",
		Disk:    map[string]string{},
		Process: map[string]string{},
		Errors:  map[string]string{},
	}

	devInfo, _ := qservice.DeviceCode.LoadFromFile()
	full.Id = devInfo.Id
	full.Name = devInfo.Name
	state := d.deviceState[devInfo.Id]
	if state.Cpu {
		full.Cpu = "alarm"
	}
	if state.Memory {
		full.Memory = "alarm"
	}
	for key, ok := range state.Disk {
		if ok {
			full.Disk[key] = "alarm"
		} else {
			full.Disk[key] = "ok"
		}
	}
	for key, ok := range state.Process {
		if ok {
			full.Process[key] = "ok"
		} else {
			full.Process[key] = "exit"
		}
	}
	second := time.Now().Local().Sub(state.Heart).Seconds()
	if second > 20 {
		full.Network = "offline"
	}
	for k, v := range state.Modules {
		// 这里应该要从日志文件里面获取
		if v.Errors != nil {
			for tp, err := range v.Errors {
				full.Errors[k] = fmt.Sprintf("%s:%s", tp, err)
			}
		}
	}

	return full, nil
}

func (a *Alarm) checkLoop() {
	// 定时检测是否有状态变化
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 查看是否有异常
			a.lock.Lock()
			a.lock.Unlock()

			for id, info := range a.deviceState {
				// 检测模块
				for name, v := range info.Modules {
					// 是否在线
					key := fmt.Sprintf("%s.%s", id, name)
					second := time.Now().Local().Sub(v.Heart).Seconds()
					a.setUploadAlarms(key, "Network", second > 20, "offline")
					// 是否有故障
					for tp, err := range v.Errors {
						a.setUploadAlarms(key, tp, err != "", err)
					}
				}
				// 检测硬件是否有故障
				key := fmt.Sprintf("%s.%s", id, "Route")
				a.setUploadAlarms(key, "Cpu", info.Cpu == true, "alarm")
				a.setUploadAlarms(key, "Memory", info.Memory == true, "alarm")
				diskAlarm := make([]string, 0)
				for k, v := range info.Disk {
					if v == true {
						diskAlarm = append(diskAlarm, fmt.Sprintf("%s:%s", strings.Trim(k, ":"), "alarm"))
					}
				}
				diskStr, _ := json.Marshal(diskAlarm)
				a.setUploadAlarms(key, "Dist", len(diskAlarm) > 0, string(diskStr))
				// 检测进程是否有故障
				processAlarm := make([]string, 0)
				for k, v := range info.Process {
					if v == false {
						processAlarm = append(processAlarm, fmt.Sprintf("%s:%s", strings.Trim(k, ":"), "exit"))
					}
				}
				processStr, _ := json.Marshal(processAlarm)
				a.setUploadAlarms(key, "Process", len(processAlarm) > 0, string(processStr))
			}

			a.uploadChan <- a.uploadState
		}
	}
}

func (a *Alarm) setUploadAlarms(key string, name string, where bool, trueValue string) {
	state := a.uploadState[key]
	if state.Alarms == nil {
		state.Alarms = map[string]string{}
	}
	if where {
		state.Alarms[name] = trueValue
	} else {
		delete(state.Alarms, name)
		if len(state.Alarms) == 0 {
			state.Alarms = nil
		}
	}
	a.uploadState[key] = state
}

func (a *Alarm) uploadLoop() {
	for true {
		select {
		case content := <-a.uploadChan:
			str, _ := json.Marshal(a.uploadState)
			// 如果和上次上报的不一致才进行上报
			if a.lastAlarm != string(str) {
				a.lastAlarm = string(str)
				if config.Config.Mode == config.ERouteServer && config.Config.UpMqtt.Addr == "" {
					// 根级别服务，则直接发送通知
					a.OnNotice("RouteDeviceAlarm", content)
				} else {
					// 向连接的Broker上报
					a.SendDeviceState(content)
				}
			}
		}
	}
}
