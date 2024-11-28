package models

type DeviceInfo struct {
	RouteUrl string       // 完整路由路径
	Id       string       // 设备码
	Name     string       // 设备名称
	Parent   string       // 父级名称
	Modules  []ModuleInfo // 包含的模块列表
	Alarms   map[string]string
}

type ModuleInfo struct {
	Name    string
	Desc    string
	Version string
}

type RouteInfo struct {
	Module  string // 模块名称
	Route   string // 方法名称
	Content any    // 入参
}

type DeviceStateFull struct {
	Id      string            // 设备码
	Name    string            // 设备名称
	Network string            // 网络情况
	Cpu     string            // CPU
	Memory  string            // 内存
	Disk    map[string]string // 磁盘
	Process map[string]string // 进程
	Errors  map[string]string // 包含的故障列表
}
