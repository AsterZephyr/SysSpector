package main

import (
	"fmt"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

var info model.SystemInfo

// 定义平台特定的函数类型
type SystemInfoFunc func(*model.SystemInfo)

// 定义平台特定的函数变量，这些变量会在平台特定的init函数中被赋值
var (
	platformGetNetworkCardInfo   SystemInfoFunc = defaultSystemInfoFunc
	platformGetGraphicsCardInfo  SystemInfoFunc = defaultSystemInfoFunc
	platformGetDisplayInfo       SystemInfoFunc = defaultSystemInfoFunc
	platformGetDynamicSystemInfo SystemInfoFunc = defaultSystemInfoFunc
)

// 默认的系统信息函数实现，当平台不支持时使用
func defaultSystemInfoFunc(_ *model.SystemInfo) {
	fmt.Println("当前平台不支持此功能")
}

func main() {
	fmt.Println("开始测试硬件信息采集功能...")

	// 测试网卡信息采集
	fmt.Println("\n===== 测试网卡信息采集 =====")
	platformGetNetworkCardInfo(&info)
	
	// 显示网卡信息
	fmt.Printf("网卡数量: %d\n", len(info.NetworkCards))
	for i, card := range info.NetworkCards {
		fmt.Printf("网卡 #%d:\n", i+1)
		fmt.Printf("  名称: %s\n", card.Name)
		fmt.Printf("  MAC地址: %s\n", card.MACAddress)
		if len(card.IPAddresses) > 0 {
			fmt.Printf("  IP地址: %s\n", card.IPAddresses[0])
		}
		fmt.Println()
	}

	// 测试显卡信息采集
	fmt.Println("\n===== 测试显卡信息采集 =====")
	platformGetGraphicsCardInfo(&info)
	
	// 显示显卡信息
	fmt.Printf("显卡数量: %d\n", len(info.GraphicsCards))
	for i, card := range info.GraphicsCards {
		fmt.Printf("显卡 #%d:\n", i+1)
		fmt.Printf("  名称: %s\n", card.Name)
		fmt.Printf("  型号: %s\n", card.Model)
		// 根据显存类型进行不同的处理
		if card.Memory != 0 {
			memoryGB := float64(card.Memory) / (1024 * 1024 * 1024)
			fmt.Printf("  内存: %.2f GB\n", memoryGB)
		}
		fmt.Println()
	}

	// 测试显示器信息采集
	fmt.Println("\n===== 测试显示器信息采集 =====")
	platformGetDisplayInfo(&info)
	
	// 显示显示器信息
	fmt.Printf("显示器数量: %d\n", len(info.Displays))
	for i, display := range info.Displays {
		fmt.Printf("显示器 #%d:\n", i+1)
		fmt.Printf("  类型: %s\n", display.Name)
		if display.Model != "" {
			fmt.Printf("  型号: %s\n", display.Model)
		}
		if display.Resolution != "" {
			fmt.Printf("  分辨率: %s\n", display.Resolution)
		}
		if display.RefreshRate > 0 {
			fmt.Printf("  刷新率: %d Hz\n", display.RefreshRate)
		}
		fmt.Println()
	}

	// 测试蓝牙信息采集
	fmt.Println("\n===== 测试蓝牙信息采集 =====")
	platformGetDynamicSystemInfo(&info)
	
	// 显示蓝牙信息
	fmt.Printf("蓝牙状态: %s\n", info.Bluetooth.Status)
	fmt.Printf("连接设备数量: %d\n", len(info.Bluetooth.Devices))
	for i, device := range info.Bluetooth.Devices {
		fmt.Printf("设备 #%d:\n", i+1)
		fmt.Printf("  名称: %s\n", device.Name)
		fmt.Printf("  地址: %s\n", device.Address)
		fmt.Printf("  类型: %s\n", device.Type)
		fmt.Println()
	}
}
