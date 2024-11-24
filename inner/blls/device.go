package blls

import (
	"encoding/json"
	"github.com/kamioair/qf/qservice"
	"router/inner/config"
	"router/inner/daos"
	"router/inner/models"
	"strconv"
	"strings"
	"sync"
)

type Device struct {
	lock            *sync.RWMutex
	UpKnockDoorFunc func(info models.DeviceInfo)
}

func NewDeviceBll() *Device {
	dev := &Device{
		lock: &sync.RWMutex{},
	}
	return dev
}

func (d *Device) NewDeviceId(devId string) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	find, err := daos.DeviceDao.GetCondition("code = ?", "id")
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
	if devId == "root" || devId == "" {
		return find.Name, nil
	}
	return devId + "_" + find.Name, nil
}

func (d *Device) KnockDoor(info models.DeviceInfo) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	find := &daos.Device{}
	if info.Parent == "" {
		find, _ = daos.DeviceDao.GetCondition("code = ?", "local")
		find.Name = info.Id
		find.Modules = d.addModules(find.Modules, info.Modules)
	} else {
		// 查找
		find, _ = daos.DeviceDao.GetCondition("code = ?", info.Id)
		if find == nil {
			// 不存在则新建
			find = &daos.Device{
				Code:    info.Id,
				Name:    info.Name,
				Parent:  info.Parent,
				Modules: d.addModules("", info.Modules),
			}
		} else {
			// 更新
			find.Name = info.Name
			find.Parent = info.Parent
			find.Modules = d.addModules(find.Modules, info.Modules)
		}
	}

	// 更新数据库
	err := daos.DeviceDao.Save(find)
	if err != nil {
		return false, err
	}

	// 如果有则向上继续敲门
	d.upKnockDoorFunc(info)

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

func (d *Device) upKnockDoorFunc(info models.DeviceInfo) {
	parent := ""
	sp := strings.Split(info.Id, "_")
	if len(sp) > 1 {
		parent = strings.Join(sp[:len(sp)-1], "_")
	} else {
		parent = "root"
	}
	newInfo := models.DeviceInfo{
		Id:      info.Id,
		Name:    info.Name,
		Parent:  parent,
		Modules: info.Modules,
	}
	d.UpKnockDoorFunc(newInfo)
}

func (d *Device) GetDeviceList() ([]models.DeviceInfo, error) {
	if daos.DeviceDao == nil {
		return nil, nil
	}

	// 获取全部列表
	list, err := daos.DeviceDao.GetAll()
	if err != nil {
		return nil, err
	}

	devInfo, _ := qservice.DeviceCode.LoadFromFile()

	okList := make([]models.DeviceInfo, 0)
	for _, m := range list {
		if m.Code == "id" {
			continue
		}

		modules := make([]models.ModuleInfo, 0)
		_ = json.Unmarshal([]byte(m.Modules), &modules)

		var dev models.DeviceInfo
		if m.Code == "local" {
			dev = models.DeviceInfo{
				Id:      m.Name,
				Name:    devInfo.Name,
				Parent:  m.Parent,
				Modules: modules,
			}
		} else {
			dev = models.DeviceInfo{
				Id:      m.Code,
				Name:    m.Name,
				Parent:  m.Parent,
				Modules: modules,
			}
		}

		okList = append(okList, dev)
	}

	return okList, nil
}

func (d *Device) GetModuleList(devCode string) (map[string]string, error) {
	finals := map[string]string{}

	// 先查找服务器的所有模块
	local, err := daos.DeviceDao.GetCondition("code = ?", "local")
	if err != nil {
		return finals, err
	}
	devInfo, _ := qservice.DeviceCode.LoadFromFile()
	modules := make([]models.ModuleInfo, 0)
	_ = json.Unmarshal([]byte(local.Modules), &modules)
	for _, m := range modules {
		finals[m.Name] = devInfo.Id
	}

	// 再查找指定设备的模块
	if devCode != "" {
		device, err := daos.DeviceDao.GetCondition("code = ?", devCode)
		if err != nil {
			return finals, err
		}
		modules := make([]models.ModuleInfo, 0)
		_ = json.Unmarshal([]byte(device.Modules), &modules)
		for _, m := range modules {
			finals[m.Name] = device.Code
		}
	}

	return finals, nil
}
