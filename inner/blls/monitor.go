package blls

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/shirou/gopsutil/v4/cpu"
	"router/inner/config"
	"time"
)

type Monitor struct {
	crn      *cron.Cron
	cpuAlarm time.Time
}

func NewMonitorBll() *Monitor {
	m := &Monitor{
		crn:      cron.New(cron.WithSeconds()),
		cpuAlarm: time.Now(),
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
	fmt.Println(err)

	m.crn.Start()
}

func (m *Monitor) checkCpu() {
	percentages, err := cpu.Percent(time.Second*5, false)
	if err != nil || len(percentages) == 0 {
		return
	}
	if percentages[0] < config.Config.Monitor.CpuAlarm.Value {
		m.cpuAlarm = time.Now()
	}
	if time.Now().Sub(m.cpuAlarm).Seconds() >= config.Config.Monitor.CpuAlarm.Duration {
		// 触发报警
		m.cpuAlarm = time.Now()
	}
}

func (m *Monitor) checkMemory() {

}
