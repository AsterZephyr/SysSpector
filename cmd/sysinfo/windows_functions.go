//go:build windows
// +build windows

package main

import (
	"github.com/AsterZephyr/SysSpector/internal/windows"
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// 在windows平台上初始化时注册实现
func init() {
	// 注册windows平台特定的实现
	getSystemInfo = windowsGetSystemInfo
}

// windowsGetSystemInfo 获取Windows系统的系统信息
func windowsGetSystemInfo() (model.SystemInfo, error) {
	return windows.GetAllSystemInfo()
}
