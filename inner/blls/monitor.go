package blls

import (
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"router/inner/config"
	"router/inner/models"
	"strings"
	"sync"
	"time"
)

type monitor struct {
	crn              *cron.Cron
	cpuAlarm         time.Time
	memAlarm         time.Time
	diskAlarm        time.Time
	lock             *sync.Mutex
	heartAlarms      map[string]time.Time
	lastCpuAlarm     string
	lastMemAlarm     string
	lastDiskAlarm    string
	lastProcessAlarm string
	lastHeartAlarm   string
	onStateNotice    func(tp string, content any)
	onHeartNotice    func(offlineIds map[string]bool)
}

func newMonitorBll(onStateNotice func(tp string, content any), onHeartNotice func(offlineIds map[string]bool)) *monitor {
	m := &monitor{
		crn:           cron.New(cron.WithSeconds()),
		cpuAlarm:      time.Now().Local(),
		memAlarm:      time.Now().Local(),
		diskAlarm:     time.Now().Local(),
		lock:          &sync.Mutex{},
		heartAlarms:   map[string]time.Time{},
		onStateNotice: onStateNotice,
		onHeartNotice: onHeartNotice,
	}
	return m
}

func (m *monitor) Start() {
	// 添加定时任务
	_, err := m.crn.AddFunc(config.Monitor.Cron, func() {
		m.checkCpu()
		m.checkMemory()
		m.checkDisk()
		m.checkProcess()
	})
	_, err = m.crn.AddFunc("0/10 * * * * ?", func() {
		m.checkHeart()
	})

	if err != nil {
		panic(err)
	}
	// 启动
	m.crn.Start()
}

func (m *monitor) AddHeart(devId string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.heartAlarms[devId] = time.Now().Local()
}

func (m *monitor) checkCpu() {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil || len(percentages) == 0 {
		return
	}

	cpuState := models.CpuMemState{}
	if percentages[0] < config.Monitor.CpuAlarm {
		m.cpuAlarm = time.Now().Local()
		cpuState = models.CpuMemState{
			Value: fmt.Sprintf("%d%", int(percentages[0])),
			IsOk:  true,
		}
	} else {
		if time.Now().Sub(m.cpuAlarm).Seconds() >= config.Monitor.Duration {
			// 触发报警
			cpuState = models.CpuMemState{
				Value: fmt.Sprintf("%d%", int(percentages[0])),
				IsOk:  false,
			}
		}
	}
	// 和上次比较，如果不一致，进行上报
	str, _ := json.Marshal(cpuState)
	if m.lastCpuAlarm != string(str) {
		m.lastCpuAlarm = string(str)
		go m.onStateNotice("CPU", cpuState)
	}
}

func (m *monitor) checkMemory() {
	v, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	memState := models.CpuMemState{}
	if v.UsedPercent < config.Monitor.MemAlarm {
		m.memAlarm = time.Now().Local()
		memState = models.CpuMemState{
			Value: fmt.Sprintf("%d%", int(v.UsedPercent)),
			IsOk:  true,
		}
	} else {
		if time.Now().Sub(m.memAlarm).Seconds() >= config.Monitor.Duration {
			// 触发报警
			memState = models.CpuMemState{
				Value: fmt.Sprintf("%d%", int(v.UsedPercent)),
				IsOk:  false,
			}
		}
	}
	// 和上次比较，如果不一致，进行上报
	str, _ := json.Marshal(memState)
	if m.lastMemAlarm != string(str) {
		m.lastMemAlarm = string(str)
		go m.onStateNotice("MEM", memState)
	}
}

func (m *monitor) checkDisk() {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return
	}

	alarms := []models.DiskState{}
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}
		if config.Config.Mode == config.ERouteServer && config.Monitor.DiskPaths == nil {
			if usage.UsedPercent >= config.Monitor.DiskAlarm {
				alarms = append(alarms, models.DiskState{
					Name:  partition.Mountpoint,
					Value: fmt.Sprintf("%d", int(usage.UsedPercent)),
					IsOk:  false,
				})
			} else {
				alarms = append(alarms, models.DiskState{
					Name:  partition.Mountpoint,
					Value: fmt.Sprintf("%d", int(usage.UsedPercent)),
					IsOk:  true,
				})
			}
		} else if config.Config.Mode == config.ERouteClient && config.Monitor.DiskPaths == nil {
			if partition.Mountpoint == "C:" {
				if usage.UsedPercent >= config.Monitor.DiskAlarm {
					alarms = append(alarms, models.DiskState{
						Name:  partition.Mountpoint,
						Value: fmt.Sprintf("%d", int(usage.UsedPercent)),
						IsOk:  false,
					})
				} else {
					alarms = append(alarms, models.DiskState{
						Name:  partition.Mountpoint,
						Value: fmt.Sprintf("%d", int(usage.UsedPercent)),
						IsOk:  true,
					})
				}
			}
		} else {
			if config.Monitor.DiskPaths == nil {
				return
			}
			exist := false
			for _, p := range config.Monitor.DiskPaths {
				if p == partition.Mountpoint {
					exist = true
					break
				}
			}
			if exist {
				if usage.UsedPercent >= config.Monitor.DiskAlarm {
					alarms = append(alarms, models.DiskState{
						Name:  partition.Mountpoint,
						Value: fmt.Sprintf("%d", int(usage.UsedPercent)),
						IsOk:  false,
					})
				} else {
					alarms = append(alarms, models.DiskState{
						Name:  partition.Mountpoint,
						Value: fmt.Sprintf("%d", int(usage.UsedPercent)),
						IsOk:  true,
					})
				}
			}
		}
	}
	// 和上次比较，如果不一致，进行上报
	str, _ := json.Marshal(alarms)
	if m.lastDiskAlarm != string(str) {
		m.lastDiskAlarm = string(str)
		m.onStateNotice("DISK", alarms)
	}
}

func (m *monitor) checkProcess() {
	processes, err := process.Processes()
	if err != nil {
		return
	}
	if config.Monitor.Processes == nil {
		return
	}

	actives := make([]models.ProcessState, 0)
	for _, p := range config.Monitor.Processes {

		exist := false
		for _, proc := range processes {
			name, err := proc.Name()
			if err != nil {
				continue // 忽略任何获取名称时出现的错误
			}

			// 比较进程名称，忽略大小写
			if strings.EqualFold(name, p) {
				exist = true
				break
			}
		}

		actives = append(actives, models.ProcessState{
			Name: p,
			IsOk: exist,
		})
	}
	// 和上次比较，如果不一致，进行上报
	str, _ := json.Marshal(actives)
	if m.lastProcessAlarm != string(str) {
		m.lastProcessAlarm = string(str)
		go m.onStateNotice("PROCESS", actives)
	}
}

func (m *monitor) checkHeart() {
	m.lock.Lock()
	defer m.lock.Unlock()

	offlineList := map[string]bool{}
	for k, v := range m.heartAlarms {
		second := time.Now().Local().Sub(v).Seconds()
		if second > 20 {
			offlineList[k] = true
		}
	}
	// 和上次比较，如果不一致，进行上报
	str, _ := json.Marshal(offlineList)
	if m.lastHeartAlarm != string(str) {
		m.lastHeartAlarm = string(str)
		go m.onHeartNotice(offlineList)
	}
}
