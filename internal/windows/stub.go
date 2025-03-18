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
	return model.SystemInfo{}, fmt.Errorf("Windows system information collection is not supported on %s", runtime.GOOS)
}
