//go:build !windows
// +build !windows

package windows

import (
	"fmt"
	"runtime"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// GetSystemInfo 是 Windows 系统信息收集的存根实现
func GetSystemInfo() (model.SystemInfo, error) {
	return model.SystemInfo{}, fmt.Errorf("windows system information collection is not supported on %s", runtime.GOOS)
}

// GetAllSystemInfo 是 Windows 系统信息收集的存根实现
func GetAllSystemInfo() (model.SystemInfo, error) {
	return model.SystemInfo{}, fmt.Errorf("windows system information collection is not supported on %s", runtime.GOOS)
}

// GetNetworkInfo 是 Windows 网络信息收集的存根实现
func GetNetworkInfo() (model.NetworkInfo, error) {
	return model.NetworkInfo{}, fmt.Errorf("windows network information collection is not supported on %s", runtime.GOOS)
}

// GetDynamicInfo 是 Windows 动态信息收集的存根实现
func GetDynamicInfo() (model.SystemInfo, error) {
	return model.SystemInfo{}, fmt.Errorf("windows dynamic information collection is not supported on %s", runtime.GOOS)
}
