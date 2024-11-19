package blls

import (
	"encoding/json"
	"router/inner/config"
	"router/inner/daos"
	"router/inner/models"
	"strconv"
	"sync"
)

type Device struct {
	lock            *sync.RWMutex
	devId           string
	UpKnockDoorFunc func(info models.DeviceInfo)
}

func NewDeviceBll(devId string) *Device {
	dev := &Device{
		lock:  &sync.RWMutex{},
		devId: devId,
	}
	return dev
}

func (d *Device) NewDeviceId() (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	find, err := daos.DeviceDao.GetCondition("code = IID")
	if err != nil {
		return "", err
	}
	id, err := strconv.Atoi(find.Name)
	if err != nil {
		id = config.Config.StartId
	}
	id++

	// 更新数据库
	find.Name = strconv.Itoa(id)
	err = daos.DeviceDao.Save(find)
	if err != nil {
		return "", err
	}

	// 格式化客户端ID并返回
	return d.devId + "-" + find.Name, nil
}

func (d *Device) KnockDoor(info models.DeviceInfo) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	parent := info.Parent
	if parent == "" {
		parent = d.devId
	}

	// 查找
	find, _ := daos.DeviceDao.GetCondition("code = ?", info.Id)
	if find == nil {
		// 不存在则新建
		find = &daos.Device{
			Code:    info.Id,
			Name:    info.Name,
			Parent:  parent,
			Modules: d.addModules("", info.Modules),
		}
	} else {
		// 更新
		find.Name = info.Name
		find.Parent = parent
		find.Modules = d.addModules(find.Modules, info.Modules)
	}

	// 更新数据库
	err := daos.DeviceDao.Save(find)
	if err != nil {
		return false, err
	}

	// 如果有则向上继续敲门
	info.Parent = parent
	d.UpKnockDoorFunc(info)

	return true, nil
}

func (d *Device) addModules(oldModulesStr string, newModules []models.ModuleInfo) string {
	finalModules := make([]models.ModuleInfo, 0)
	if oldModulesStr != "" {
		_ = json.Unmarshal([]byte(oldModulesStr), &finalModules)
	}
	// 遍历
	for _, nm := range newModules {
		exist := false
		for _, om := range finalModules {
			if om.Name == nm.Name {
				om.Desc = nm.Desc
				om.Version = nm.Version
				exist = true
			}
		}
		if exist == false {
			finalModules = append(finalModules, nm)
		}
	}
	js, _ := json.Marshal(finalModules)
	return string(js)
}
