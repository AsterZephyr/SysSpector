//go:build darwin
// +build darwin

package main

import (
	"github.com/AsterZephyr/SysSpector/internal/darwin"
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// 在darwin平台上初始化时注册实现
func init() {
	// 注册darwin平台特定的实现
	getSystemInfo = darwinGetSystemInfo
}

// darwinGetSystemInfo 获取macOS系统的系统信息
func darwinGetSystemInfo() (model.SystemInfo, error) {
	return darwin.GetSystemInfo()
}
