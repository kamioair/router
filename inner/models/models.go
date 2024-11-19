package models

type DeviceInfo struct {
	Id      string       // 设备码
	Name    string       // 设备名称
	Parent  string       // 父级名称
	Modules []ModuleInfo // 包含的模块列表
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
