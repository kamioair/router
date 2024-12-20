package models

import "sort"

// DeviceKnock 设备敲门信息
type DeviceKnock struct {
	Id      string           // 设备码
	Name    string           // 设备名称
	FullUrl string           // 完整路径
	Modules ModuleCollection // 包含的模块列表
}

// DeviceInfo 完整设备信息
type DeviceInfo struct {
	Id       string           // 设备码
	Name     string           // 设备名称
	FullUrl  string           // 完整路径
	Parent   string           // 父级名称
	IsOnline bool             // 网络是否在线
	Cpu      CpuMemState      // CPU
	Memory   CpuMemState      // 内存
	Disk     []DiskState      // 磁盘
	Process  []ProcessState   // 进程
	Modules  ModuleCollection // 包含的模块列表
}

// ModuleCollection 模块列表
type ModuleCollection []ModuleInfo

func (c *ModuleCollection) Add(list []ModuleInfo) {
	for _, nm := range list {
		exist := false
		for i, om := range *c {
			if om.Name == nm.Name {
				(*c)[i].Desc = nm.Desc
				(*c)[i].Version = nm.Version
				exist = true
				break
			}
		}
		if exist == false {
			(*c) = append((*c), nm)
		}
	}
}

// DeviceAlarm 设备报警信息
type DeviceAlarm struct {
	Id      string // 设备码
	Name    string // 设备名称
	Parent  string // 父级名称
	FullUrl string // 完整路由路径
	Alarms  []Item // 包含的警报列表
}

func (da *DeviceAlarm) Set(name string, alarmWhere bool, alarmValue string, dev DeviceInfo) {
	da.Id = dev.Id
	da.Name = dev.Name
	da.Parent = dev.Parent
	da.FullUrl = dev.FullUrl
	if alarmWhere {
		// 添加到列表
		add := true
		for i := 0; i < len(da.Alarms); i++ {
			if da.Alarms[i].Name == name {
				da.Alarms[i].Value = alarmValue
				add = false
				break
			}
		}
		if add {
			da.Alarms = append(da.Alarms, Item{
				Name:  name,
				Value: alarmValue,
			})
			sort.Slice(da.Alarms, func(i, j int) bool {
				return i > j
			})
		}
	} else {
		// 从列表中移除
		index := -1
		for i := 0; i < len(da.Alarms); i++ {
			if da.Alarms[i].Name == name {
				index = i
				break
			}
		}
		if index >= 0 {
			da.Alarms = append(da.Alarms[:index], da.Alarms[index+1:]...)
		}
	}
}

// CpuMemState CPU和内存状态
type CpuMemState struct {
	Value string // 当前百分比
	IsOk  bool   // 是否正常
}

// DiskState 硬盘状态
type DiskState struct {
	Name  string // 盘符名称
	Value string // 当前百分比
	IsOk  bool   // 是否正常
}

// ProcessState 进程状态
type ProcessState struct {
	Name string // 进程名称
	IsOk bool   // 是否正常
}

// Item 其他项目内容
type Item struct {
	Name  string
	Value string
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	Name    string // 模块名称
	Desc    string // 模块描述
	Version string // 模块版本
	Error   string // 详细错误信息（从文件中读取）
}

// RouteInfo 路由信息
type RouteInfo struct {
	Module  string // 模块名称
	Route   string // 方法名称
	Content any    // 入参
}
