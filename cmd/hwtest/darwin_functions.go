//go:build darwin
// +build darwin

package main

import (
	"log"

	"github.com/AsterZephyr/SysSpector/internal/darwin"
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// 在darwin平台上初始化时注册实现
func init() {
	// 注册darwin平台特定的实现
	platformGetNetworkCardInfo = darwinGetNetworkCardInfo
	platformGetGraphicsCardInfo = darwinGetGraphicsCardInfo
	platformGetDisplayInfo = darwinGetDisplayInfo
	platformGetDynamicSystemInfo = darwinGetDynamicSystemInfo
}

// darwinGetNetworkCardInfo 获取macOS系统的网卡信息
func darwinGetNetworkCardInfo(info *model.SystemInfo) {
	err := darwin.GetNetworkCardInfo(info)
	if err != nil {
		log.Fatalf("采集网卡信息失败: %v", err)
	}
}

// darwinGetGraphicsCardInfo 获取macOS系统的显卡信息
func darwinGetGraphicsCardInfo(info *model.SystemInfo) {
	err := darwin.GetGraphicsCardInfo(info)
	if err != nil {
		log.Fatalf("采集显卡信息失败: %v", err)
	}
}

// darwinGetDisplayInfo 获取macOS系统的显示器信息
func darwinGetDisplayInfo(info *model.SystemInfo) {
	err := darwin.GetDisplayInfo(info)
	if err != nil {
		log.Fatalf("采集显示器信息失败: %v", err)
	}
}

// darwinGetDynamicSystemInfo 获取macOS系统的动态信息
func darwinGetDynamicSystemInfo(info *model.SystemInfo) {
	err := darwin.GetDynamicSystemInfo(info)
	if err != nil {
		log.Fatalf("采集蓝牙信息失败: %v", err)
	}
}
