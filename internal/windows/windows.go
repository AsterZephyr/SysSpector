//go:build windows
// +build windows

package windows

import (
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// GetAllSystemInfo 获取所有Windows系统信息
func GetAllSystemInfo() (model.SystemInfo, error) {
	// 获取基本系统信息
	sysInfo, err := GetSystemInfo()
	if err != nil {
		return sysInfo, err
	}
	
	// 获取网络信息
	netInfo, err := GetNetworkInfo()
	if err == nil {
		// 将网络信息整合到系统信息中
		sysInfo.Network = netInfo
	}
	
	// 获取动态信息
	dynamicInfo, err := GetDynamicInfo()
	if err == nil {
		sysInfo.DiskUsage = dynamicInfo.DiskUsage
		sysInfo.MemoryUsage = dynamicInfo.MemoryUsage
		sysInfo.Battery = dynamicInfo.Battery
		sysInfo.ACAdapter = dynamicInfo.ACAdapter
		sysInfo.Bluetooth = dynamicInfo.Bluetooth
		sysInfo.Temperature = dynamicInfo.Temperature
		sysInfo.InstalledApps = dynamicInfo.InstalledApps
		sysInfo.RunningApps = dynamicInfo.RunningApps
		sysInfo.UpTime = dynamicInfo.UpTime
	}
	
	return sysInfo, nil
}
