package blls

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/kamioair/qf/qservice"
	"router/inner/daos"
	"router/inner/models"
	"strings"
	"sync"
)

type Device struct {
	UpKnockDoorFunc func(info models.DeviceInfo)
	lock            *sync.RWMutex
	localCaches     models.DeviceInfo
	deviceCaches    map[string]models.DeviceInfo
	GetAlarmsFunc   func() map[string]map[string]string
}

func NewDeviceBll() *Device {
	dev := &Device{
		lock:         &sync.RWMutex{},
		deviceCaches: make(map[string]models.DeviceInfo),
	}

	return dev
}

func (d *Device) NewDeviceId() (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// 生成一个新的ID
	newId := uuid.NewString()

	// 返回新的ID
	return newId, nil
}

func (d *Device) KnockDoor(info models.DeviceInfo, devId string) (any, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	find := &daos.Device{}
	if info.Parent == "" || info.Id == devId {
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

	d.deviceCaches[info.Id] = info

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
	newInfo := models.DeviceInfo{
		Id:      info.Id,
		Name:    info.Name,
		Parent:  info.Parent,
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

		for k, v := range d.GetAlarmsFunc() {
			sp := strings.Split(k, ".")
			if sp[0] != dev.Id {
				continue
			}
			dev.Alarms = v
		}

		okList = append(okList, dev)
	}

	deviceMap := buildDeviceMap(okList)
	for i := range okList {
		okList[i].RouteUrl = buildFullPath(deviceMap, &okList[i])
	}

	return okList, nil
}

func buildDeviceMap(devices []models.DeviceInfo) map[string]*models.DeviceInfo {
	deviceMap := make(map[string]*models.DeviceInfo)
	for i := range devices {
		deviceMap[devices[i].Id] = &devices[i]
	}
	return deviceMap
}

func buildFullPath(deviceMap map[string]*models.DeviceInfo, device *models.DeviceInfo) string {
	// 如果没有父节点，直接返回当前节点的Id
	if device.Parent == "" {
		return device.Id
	}

	// 递归获取父节点的完整路径
	parentNode, exists := deviceMap[device.Parent]
	if exists {
		return buildFullPath(deviceMap, parentNode) + "/" + device.Id
	}

	// 如果找不到父节点，返回当前节点的Id
	return device.Id
}

func (d *Device) GetModuleList(devCodes []string) (map[string]string, error) {
	finals := map[string]string{}

	// 先查找服务器的所有模块
	for _, devCode := range devCodes {
		if devCode == "local" {
			local, err := daos.DeviceDao.GetCondition("code = ?", devCode)
			if err != nil {
				return finals, err
			}
			devInfo, _ := qservice.DeviceCode.LoadFromFile()
			modules := make([]models.ModuleInfo, 0)
			_ = json.Unmarshal([]byte(local.Modules), &modules)
			for _, m := range modules {
				finals[m.Name] = devInfo.Id
			}
		} else {
			// 再查找指定设备的模块
			device, err := daos.DeviceDao.GetCondition("code = ?", devCode)
			if err != nil {
				return finals, err
			}
			if device != nil {
				modules := make([]models.ModuleInfo, 0)
				_ = json.Unmarshal([]byte(device.Modules), &modules)
				for _, m := range modules {
					finals[m.Name] = device.Code
				}
			}
		}
	}

	return finals, nil
}
