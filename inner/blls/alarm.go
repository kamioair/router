package blls

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Alarm struct {
	moduleAlarms map[string]map[string]moduleAlarm
	deviceAlarm  map[string]deviceAlarm
	lock         *sync.RWMutex
}

type moduleAlarm struct {
	Heart  time.Time
	Alarms []alarmItem
}

type alarmItem struct {
	Name  string
	Value string
}

type deviceAlarm struct {
	Cpu     string
	Memory  string
	Disk    []alarmItem
	Process []alarmItem
}

func NewAlarmBll() *Alarm {
	a := &Alarm{
		lock:         &sync.RWMutex{},
		moduleAlarms: map[string]map[string]moduleAlarm{},
		deviceAlarm:  map[string]deviceAlarm{},
	}

	go a.checkLoop()
	return a
}

func (a *Alarm) AddHeart(key string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	sp := strings.Split(key, "^")
	if len(sp) < 2 {
		return
	}

	if a.moduleAlarms[sp[0]] == nil {
		a.moduleAlarms[sp[0]] = map[string]moduleAlarm{}
	}
	m := a.moduleAlarms[sp[0]][sp[1]]
	m.Heart = time.Now()
	a.moduleAlarms[sp[0]][sp[1]] = m
}

func (a *Alarm) AddError(key string, title, err string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	sp := strings.Split(key, "^")
	if len(sp) < 2 {
		return
	}

	if a.moduleAlarms[sp[0]] == nil {
		a.moduleAlarms[sp[0]] = map[string]moduleAlarm{}
	}
	m := a.moduleAlarms[sp[0]][sp[1]]
	if m.Alarms == nil {
		m.Alarms = make([]alarmItem, 0)
	}
	m.Alarms = append(m.Alarms, alarmItem{
		Name:  title,
		Value: err,
	})
}

func (a *Alarm) AddAlarmCpu(devCode string, alarm bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	m := a.deviceAlarm[devCode]
	m.Cpu = "ok"
	if alarm {
		m.Cpu = "alarm"
	}
	a.deviceAlarm[devCode] = m
}

func (a *Alarm) AddAlarmMemory(devCode string, alarm bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	m := a.deviceAlarm[devCode]
	m.Memory = "ok"
	if alarm {
		m.Memory = "alarm"
	}
	a.deviceAlarm[devCode] = m
}

func (a *Alarm) AddAlarmDisk(devCode string, alarm map[string]bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	m := a.deviceAlarm[devCode]
	m.Disk = make([]alarmItem, 0)
	for k, v := range alarm {
		value := "ok"
		if v {
			value = "alarm"
		}
		m.Disk = append(m.Disk, alarmItem{
			Name:  k,
			Value: value,
		})
	}
	a.deviceAlarm[devCode] = m
}

func (a *Alarm) AddAlarmProcess(devCode string, actives map[string]bool) {
	a.lock.Lock()
	defer a.lock.Unlock()

	m := a.deviceAlarm[devCode]
	m.Process = make([]alarmItem, 0)
	for k, v := range actives {
		value := "running"
		if v == false {
			value = "exit"
		}
		m.Process = append(m.Process, alarmItem{
			Name:  k,
			Value: value,
		})
	}
	a.deviceAlarm[devCode] = m
}

func (a *Alarm) checkLoop() {
	// 定时检测是否有状态变化
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 查看是否有异常
			a.lock.Lock()
			a.lock.Unlock()

			fmt.Println(a.moduleAlarms)
			fmt.Println(a.deviceAlarm)
		}
	}
}
