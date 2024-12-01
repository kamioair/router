package daos

import "github.com/kamioair/qf/qdefine"

type Device struct {
	qdefine.DbFull
	Code    string `gorm:"unique"` // 设备码
	Name    string // 设备名称
	Parent  string // 父级设备码
	Modules string // 包含的模块列表 Json
}
