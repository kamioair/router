package daos

import (
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qdb"
	"router/inner/config"
	"strconv"
)

var (
	DeviceDao *qdefine.BaseDao[Device]
)

func Init(module string) {
	db := qdb.NewDb(module)

	// 初始化
	DeviceDao = qdefine.NewDao[Device](db)
	// 写入两条固定记录
	if DeviceDao.GetCount() == 0 {
		_ = DeviceDao.Create(&Device{Code: "IID", Name: strconv.Itoa(config.Config.StartId)})
		_ = DeviceDao.Create(&Device{Code: "LocalId"})
	}
}
