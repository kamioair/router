package models

type RouteInfo struct {
	Module  string // 模块名称
	Route   string // 方法名称
	Content any    // 入参
}

type ServerInfo struct {
	DeviceCode string       // 服务端设备码
	Modules    []ModuleInfo // 服务所有模块列表
}

type ModuleInfo struct {
	Name    string
	Desc    string
	Version string
}
