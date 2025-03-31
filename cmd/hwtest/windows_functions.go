//go:build windows
// +build windows

package main

import (
	"log"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// 在windows平台上初始化时注册实现
func init() {
	// 注册windows平台特定的实现
	platformGetNetworkCardInfo = windowsGetNetworkCardInfo
	platformGetGraphicsCardInfo = windowsGetGraphicsCardInfo
	platformGetDisplayInfo = windowsGetDisplayInfo
	platformGetDynamicSystemInfo = windowsGetDynamicSystemInfo
}

// windowsGetNetworkCardInfo 获取Windows系统的网卡信息
func windowsGetNetworkCardInfo(info *model.SystemInfo) {
	// 暂未实现
	log.Println("Windows系统暂不支持网卡信息采集")
}

// windowsGetGraphicsCardInfo 获取Windows系统的显卡信息
func windowsGetGraphicsCardInfo(info *model.SystemInfo) {
	// 暂未实现
	log.Println("Windows系统暂不支持显卡信息采集")
}

// windowsGetDisplayInfo 获取Windows系统的显示器信息
func windowsGetDisplayInfo(info *model.SystemInfo) {
	// 暂未实现
	log.Println("Windows系统暂不支持显示器信息采集")
}

// windowsGetDynamicSystemInfo 获取Windows系统的动态信息
func windowsGetDynamicSystemInfo(info *model.SystemInfo) {
	// 暂未实现
	log.Println("Windows系统暂不支持蓝牙信息采集")
}
