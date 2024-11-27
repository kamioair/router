package blls

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"router/inner/config"
	"strings"
	"time"
)

type Monitor struct {
	crn             *cron.Cron
	cpuAlarm        time.Time
	memAlarm        time.Time
	diskAlarm       time.Time
	AddAlarmCpu     func(alarm bool)
	AddAlarmMem     func(alarm bool)
	AddAlarmDisk    func(alarm map[string]bool)
	AddAlarmProcess func(actives map[string]bool)
}

func NewMonitorBll() *Monitor {
	m := &Monitor{
		crn:       cron.New(cron.WithSeconds()),
		cpuAlarm:  time.Now().Local(),
		memAlarm:  time.Now().Local(),
		diskAlarm: time.Now().Local(),
	}
	return m
}

func (m *Monitor) Start() {
	// 添加定时任务
	_, err := m.crn.AddFunc(config.Config.Monitor.Cron, func() {
		m.checkCpu()
	})
	_, err = m.crn.AddFunc(config.Config.Monitor.Cron, func() {
		m.checkMemory()
	})
	_, err = m.crn.AddFunc(config.Config.Monitor.Cron, func() {
		m.checkDisk()
	})
	_, err = m.crn.AddFunc(config.Config.Monitor.Cron, func() {
		m.checkProcess()
	})
	fmt.Println(err)

	m.crn.Start()
}

func (m *Monitor) checkCpu() {
	percentages, err := cpu.Percent(time.Second*5, false)
	if err != nil || len(percentages) == 0 {
		return
	}
	if percentages[0] < config.Config.Monitor.CpuAlarm {
		m.cpuAlarm = time.Now().Local()
	}
	if time.Now().Sub(m.cpuAlarm).Seconds() >= config.Config.Monitor.Duration {
		// 触发报警
		m.cpuAlarm = time.Now().Local()
		m.AddAlarmCpu(true)
	} else {
		m.AddAlarmCpu(false)
	}
}

func (m *Monitor) checkMemory() {
	v, err := mem.VirtualMemory()
	if err != nil {
		return
	}
	if v.UsedPercent < config.Config.Monitor.MemAlarm {
		m.memAlarm = time.Now().Local()
	}
	if time.Now().Sub(m.memAlarm).Seconds() >= config.Config.Monitor.Duration {
		// 触发报警
		m.memAlarm = time.Now().Local()
		m.AddAlarmMem(true)
	} else {
		m.AddAlarmMem(false)
	}
}

func (m *Monitor) checkDisk() {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return
	}

	alarms := map[string]bool{}
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}
		if config.Config.Mode == config.ERouteServer && config.Config.Monitor.DiskPaths == nil {
			if usage.UsedPercent >= config.Config.Monitor.DiskAlarm {
				alarms[partition.Mountpoint] = true
			} else {
				alarms[partition.Mountpoint] = false
			}
		} else if config.Config.Mode == config.ERouteClient && config.Config.Monitor.DiskPaths == nil {
			if partition.Mountpoint == "C:" {
				if usage.UsedPercent >= config.Config.Monitor.DiskAlarm {
					alarms[partition.Mountpoint] = true
				} else {
					alarms[partition.Mountpoint] = false
				}
			}
		} else {
			if config.Config.Monitor.DiskPaths == nil {
				return
			}
			exist := false
			for _, p := range config.Config.Monitor.DiskPaths {
				if p == partition.Mountpoint {
					exist = true
					break
				}
			}
			if exist {
				if usage.UsedPercent >= config.Config.Monitor.DiskAlarm {
					alarms[partition.Mountpoint] = true
				} else {
					alarms[partition.Mountpoint] = false
				}
			}
		}
	}
	m.AddAlarmDisk(alarms)
}

func (m *Monitor) checkProcess() {
	processes, err := process.Processes()
	if err != nil {
		return
	}
	if config.Config.Monitor.Processes == nil {
		return
	}

	actives := map[string]bool{}
	for _, p := range config.Config.Monitor.Processes {

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

		actives[p] = exist
	}

	m.AddAlarmProcess(actives)
}
