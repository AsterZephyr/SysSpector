package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/AsterZephyr/SysSpector/internal/darwin"
	"github.com/AsterZephyr/SysSpector/internal/windows"
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

func main() {
	// 设置日志输出到标准错误
	log.SetOutput(os.Stderr)
	log.Println("Starting system information collection...")

	var sysInfo model.SystemInfo
	var err error

	// 根据操作系统类型选择相应的系统信息收集方法
	switch runtime.GOOS {
	case "windows":
		// Windows 系统
		log.Println("Detected Windows OS")
		sysInfo, err = windows.GetSystemInfo()
		if err != nil {
			log.Printf("Error collecting Windows system information: %v", err)
			// 在Windows上，即使有错误也继续执行，显示已收集到的信息
			fmt.Println("Some system information could not be collected. Displaying available information:")
		}
	case "darwin":
		// macOS 系统
		log.Println("Detected macOS")
		sysInfo, err = darwin.GetSystemInfo()
		if err != nil {
			log.Fatalf("Error collecting macOS system information: %v", err)
		}
	default:
		// 不支持的操作系统
		log.Fatalf("Unsupported operating system: %s", runtime.GOOS)
	}

	// 以格式化的方式打印系统信息
	printSystemInfo(sysInfo)

	// 如果命令行参数中包含 --save，则将系统信息保存到文件
	if len(os.Args) > 1 && os.Args[1] == "--save" {
		outputFile := "sysinfo.txt"
		if len(os.Args) > 2 {
			// 如果提供了文件名，则使用提供的文件名
			outputFile = os.Args[2]
		}

		// 格式化输出内容
		output := formatSystemInfo(sysInfo)

		// 写入文件
		err = os.WriteFile(outputFile, []byte(output), 0644)
		if err != nil {
			log.Fatalf("Error writing to file %s: %v", outputFile, err)
		}
		log.Printf("System information saved to %s", outputFile)
	}

	// 在Windows系统上，程序结束前暂停，等待用户按键
	if runtime.GOOS == "windows" {
		fmt.Println("\nPress Enter to exit...")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
	}
}

// printSystemInfo 格式化输出系统信息
func printSystemInfo(info model.SystemInfo) {
	// 硬件基础数据
	fmt.Println("======================= 硬件基础数据 =======================")
	fmt.Printf("%-20s %-20s %s\n", "主机名", "", info.Hostname)
	fmt.Printf("%-20s %-20s %s\n", "操作系统", "", info.OS)
	fmt.Printf("%-20s %-20s %s\n", "系统版本", "", info.SystemVersion)
	fmt.Printf("%-20s %-20s %s\n", "电脑名称", "", info.ComputerName)
	fmt.Printf("%-20s %-20s %s\n", "型号名称", "", info.Model)
	if info.ModelID != "" {
		fmt.Printf("%-20s %-20s %s\n", "型号标识符", "", info.ModelID)
	}
	fmt.Printf("%-20s %-20s %s\n", "序列号", "", info.SerialNumber)
	fmt.Printf("%-20s %-20s %s\n", "硬件UUID", "", info.UUID)
	fmt.Printf("%-20s %-20s %s\n", "处理器名称", "", info.CPU.Model)
	fmt.Printf("%-20s %-20s %d\n", "CPU核心数", "", info.CPU.Cores)
	fmt.Printf("%-20s %-20s %.2f GB\n", "内存", "", float64(info.Memory.Total)/(1024*1024*1024))
	fmt.Printf("%-20s %-20s %s\n", "内存类型", "", info.Memory.Type)
	
	// 显示WiFi支持的PHY模式
	if info.Network.WiFi.SupportedPHY != "" {
		fmt.Printf("%-20s %-20s %s\n", "WiFi支持的PHY模式", "", info.Network.WiFi.SupportedPHY)
	}
	
	// 显示硬盘容量
	var maxDiskSize uint64
	// 检查 info.Disks 中的磁盘大小
	for _, disk := range info.Disks {
		// 如果 disk.Size 小于 1000，可能是以 GB 为单位
		if disk.Size < 1000 && disk.Size > 0 {
			// 转换为字节
			sizeInBytes := disk.Size * 1024 * 1024 * 1024
			if sizeInBytes > maxDiskSize {
				maxDiskSize = sizeInBytes
			}
		} else if disk.Size > maxDiskSize {
			maxDiskSize = disk.Size
		}
	}
	
	// 如果从 info.Disks 中找不到有效的磁盘大小，则使用 info.DiskUsage
	if maxDiskSize == 0 && len(info.DiskUsage) > 0 {
		// 查找根分区或容量最大的分区
		var maxPartitionSize uint64
		var rootPartitionSize uint64
		var hasRootPartition bool
		
		for _, partition := range info.DiskUsage {
			if partition.MountPoint == "/" {
				rootPartitionSize = partition.Total
				hasRootPartition = true
			}
			if partition.Total > maxPartitionSize {
				maxPartitionSize = partition.Total
			}
		}
		
		// 优先使用根分区，其次使用最大分区
		if hasRootPartition {
			maxDiskSize = rootPartitionSize
		} else {
			maxDiskSize = maxPartitionSize
		}
	}
	
	// 显示硬盘容量
	if maxDiskSize > 0 {
		diskSizeGB := float64(maxDiskSize) / (1024 * 1024 * 1024)
		fmt.Printf("%-20s %-20s %.2f GB\n", "硬盘容量", "", diskSizeGB)
	} else {
		fmt.Printf("%-20s %-20s %s\n", "硬盘容量", "", "未知")
	}

	// 硬件动态数据
	fmt.Println("\n======================= 硬件动态数据 =======================")
	
	// 显示硬盘使用情况
	if len(info.DiskUsage) > 0 {
		var totalUsed uint64
		for _, partition := range info.DiskUsage {
			totalUsed += partition.Used
		}
		usedGB := float64(totalUsed) / (1024 * 1024 * 1024)
		fmt.Printf("%-20s %-20s %.2f GB\n", "硬盘容量（已使用）", "", usedGB)
	}
	
	// 显示内存使用情况
	fmt.Printf("%-20s %-20s %.2f GB\n", "内存容量（已使用）", "", float64(info.MemoryUsage.Used)/(1024*1024*1024))
	
	// 显示电池信息
	if info.Battery.IsPresent {
		fmt.Printf("%-20s %-20s %d%%\n", "电量信息", "", info.Battery.Percentage)
		if info.Battery.IsCharging {
			fmt.Printf("%-20s %-20s %s\n", "正在充电", "", "是")
		} else {
			fmt.Printf("%-20s %-20s %s\n", "正在充电", "", "否")
		}
		
		// 电池电量低于20%为警告水平
		if info.Battery.Percentage < 20 {
			fmt.Printf("%-20s %-20s %s\n", "电池电量低于警告水平", "", "是")
		} else {
			fmt.Printf("%-20s %-20s %s\n", "电池电量低于警告水平", "", "否")
		}
		
		fmt.Printf("%-20s %-20s %d\n", "循环计数", "", info.Battery.CycleCount)
		if info.Battery.Health != "" {
			fmt.Printf("%-20s %-20s %s\n", "电池状态", "", info.Battery.Health)
		} else if info.Battery.Status != "" {
			fmt.Printf("%-20s %-20s %s\n", "电池状态", "", info.Battery.Status)
		}
		
		if info.Battery.TimeRemaining > 0 {
			hours := info.Battery.TimeRemaining / 60
			minutes := info.Battery.TimeRemaining % 60
			fmt.Printf("%-20s %-20s %d小时%d分钟\n", "剩余使用时间", "", hours, minutes)
		}
	}
	
	// 显示交流充电器信息
	if info.ACAdapter.Connected {
		fmt.Printf("%-20s %-20s %s\n", "交流充电器-连接状态", "", "已连接")
		if info.ACAdapter.SerialNum != "" {
			fmt.Printf("%-20s %-20s %s\n", "交流充电器-序列号", "", info.ACAdapter.SerialNum)
		}
		if info.ACAdapter.Name != "" {
			fmt.Printf("%-20s %-20s %s\n", "交流充电器-名称", "", info.ACAdapter.Name)
		}
		if info.ACAdapter.Wattage > 0 {
			fmt.Printf("%-20s %-20s %dW\n", "交流充电器-功率", "", info.ACAdapter.Wattage)
		}
	} else {
		fmt.Printf("%-20s %-20s %s\n", "交流充电器-连接状态", "", "未连接")
	}
	
	// 显示蓝牙信息
	if info.Bluetooth.Enabled {
		fmt.Printf("%-20s %-20s %s\n", "蓝牙-状态", "", "打开")
		
		// 显示已连接的蓝牙设备
		connectedDevices := []string{}
		for _, device := range info.Bluetooth.Devices {
			if device.Connected {
				connectedDevices = append(connectedDevices, device.Name)
			}
		}
		
		if len(connectedDevices) > 0 {
			devicesList := strings.Join(connectedDevices, "、")
			fmt.Printf("%-20s %-20s %s\n", "蓝牙-连接设备", "", devicesList)
		} else {
			fmt.Printf("%-20s %-20s %s\n", "蓝牙-连接设备", "", "未找到已连接设备")
		}
	} else {
		fmt.Printf("%-20s %-20s %s\n", "蓝牙-状态", "", "关闭")
	}
	
	// 显示温度信息
	if len(info.Temperature) > 0 {
		fmt.Printf("%-20s\n", "设备温度")
		for _, sensor := range info.Temperature {
			fmt.Printf("  %-18s %-20s %.1f°C\n", sensor.Name, "", sensor.Temperature)
		}
	}
	
	// 显示WiFi自动连接状态
	if info.WiFiAutoJoin.IsConfigured {
		fmt.Printf("%-20s %-20s %s\n", "无线Wi-Fi自动连接状态", "", info.WiFiAutoJoin.Status)
		if len(info.WiFiAutoJoin.Networks) > 0 {
			fmt.Printf("%-20s\n", "自动连接的网络")
			for i, network := range info.WiFiAutoJoin.Networks {
				if network.AutoJoin {
					fmt.Printf("  %-18s %-20s %s\n", fmt.Sprintf("%d", i+1), "", network.SSID)
				}
			}
		}
	}

	// 网络客户端动态数据
	fmt.Println("\n======================= 网络客户端动态数据 =======================")
	
	// 显示WiFi信息
	if info.Network.WiFi.IsConnected {
		fmt.Printf("%-20s\n", "WiFi信息")
		fmt.Printf("  %-18s %-20s %s\n", "SSID", "", info.Network.WiFi.SSID)
		fmt.Printf("  %-18s %-20s %d dBm\n", "信号强度", "", info.Network.WiFi.RSSI)
		fmt.Printf("  %-18s %-20s %s\n", "PHY模式", "", info.Network.WiFi.PHYMode)
		fmt.Printf("  %-18s %-20s %d\n", "频道", "", info.Network.WiFi.Channel)
		fmt.Printf("  %-18s %-20s %.2f GHz\n", "频率", "", info.Network.WiFi.Frequency)
		fmt.Printf("  %-18s %-20s %d Mbps\n", "传输速率", "", info.Network.WiFi.TxRate)
		fmt.Printf("\n")
	}
	
	// 显示公网IP
	if info.Network.PublicIP != "" {
		fmt.Printf("%-20s %-20s %s\n", "公网IP", "", info.Network.PublicIP)
		fmt.Printf("\n")
	}
	
	// 显示DNS服务器
	if len(info.Network.DNS.Servers) > 0 {
		fmt.Printf("%-20s\n", "DNS服务器")
		for i, server := range info.Network.DNS.Servers {
			fmt.Printf("  %-18s %-20s %s\n", fmt.Sprintf("%d", i+1), "", server)
		}
		fmt.Printf("\n")
	}
	
	// 显示VPN信息
	if info.Network.VPN.IsConnected {
		fmt.Printf("%-20s\n", "VPN信息")
		fmt.Printf("  %-18s %-20s %s\n", "状态", "", "已连接")
		if info.Network.VPN.Provider != "" {
			fmt.Printf("  %-18s %-20s %s\n", "提供商", "", info.Network.VPN.Provider)
		}
		if info.Network.VPN.NodeName != "" {
			fmt.Printf("  %-18s %-20s %s\n", "节点名称", "", info.Network.VPN.NodeName)
		}
		fmt.Printf("\n")
	}
	
	// 显示网络延迟信息
	if len(info.Network.Latency.Targets) > 0 {
		fmt.Printf("%-20s\n", "网络延迟信息")
		for _, target := range info.Network.Latency.Targets {
			fmt.Printf("  %-18s %-20s %s\n", "目标", "", target.TargetName)
			fmt.Printf("    %-16s %-20s %.2f ms\n", "最小延迟", "", target.MinLatency)
			fmt.Printf("    %-16s %-20s %.2f ms\n", "平均延迟", "", target.AvgLatency)
			fmt.Printf("    %-16s %-20s %.2f ms\n", "最大延迟", "", target.MaxLatency)
			fmt.Printf("    %-16s %-20s %.1f%%\n", "丢包率", "", target.PacketLoss)
			fmt.Printf("\n")
		}
	}

	// 显示蓝牙信息
	if info.Bluetooth.IsAvailable {
		fmt.Println("蓝牙信息:")
		fmt.Printf("  状态: %s\n", info.Bluetooth.Status)
		fmt.Printf("  名称: %s\n", info.Bluetooth.Name)
		fmt.Printf("  地址: %s\n", info.Bluetooth.Address)

		if len(info.Bluetooth.ConnectedDevices) > 0 {
			fmt.Println("  已连接设备:")
			for _, device := range info.Bluetooth.ConnectedDevices {
				fmt.Printf("    - %s (%s)\n", device.Name, device.Type)
			}
		}
	}

	// 显示WiFi自动连接状态
	if info.WiFiAutoJoin.IsConfigured {
		fmt.Println("WiFi自动连接:")
		fmt.Printf("  状态: %s\n", info.WiFiAutoJoin.Status)
		if len(info.WiFiAutoJoin.Networks) > 0 {
			fmt.Println("  已保存网络:")
			for _, network := range info.WiFiAutoJoin.Networks {
				fmt.Printf("    - %s (自动连接: %v)\n", network.SSID, network.AutoJoin)
			}
		}
	}

	// 显示已安装应用
	if len(info.InstalledApps) > 0 {
		fmt.Printf("\n%-20s\n", "已安装应用")
		displayCount := 10 // 只显示前10个应用
		if len(info.InstalledApps) < displayCount {
			displayCount = len(info.InstalledApps)
		}
		for i := 0; i < displayCount; i++ {
			app := info.InstalledApps[i]
			if app.Version != "" {
				fmt.Printf("  %-18s %-20s %s (版本: %s)\n", fmt.Sprintf("%d", i+1), "", app.Name, app.Version)
			} else {
				fmt.Printf("  %-18s %-20s %s\n", fmt.Sprintf("%d", i+1), "", app.Name)
			}
		}
		if len(info.InstalledApps) > displayCount {
			fmt.Printf("  %-18s %-20s %s\n", "", "", fmt.Sprintf("... 还有 %d 个应用 ...", len(info.InstalledApps)-displayCount))
		}
	}
	
	// 显示正在运行的应用
	if len(info.RunningApps) > 0 {
		fmt.Printf("\n%-20s\n", "正在运行的应用")
		displayCount := 10 // 只显示前10个进程
		if len(info.RunningApps) < displayCount {
			displayCount = len(info.RunningApps)
		}
		for i := 0; i < displayCount; i++ {
			proc := info.RunningApps[i]
			fmt.Printf("  %-18s %-20s %s (PID: %d)\n", fmt.Sprintf("%d", i+1), "", proc.Name, proc.PID)
			fmt.Printf("    %-16s %-20s %.1f%%\n", "CPU", "", proc.CPU)
			fmt.Printf("    %-16s %-20s %.2f MB\n", "内存", "", float64(proc.Memory)/(1024*1024))
		}
		if len(info.RunningApps) > displayCount {
			fmt.Printf("  %-18s %-20s %s\n", "", "", fmt.Sprintf("... 还有 %d 个进程 ...", len(info.RunningApps)-displayCount))
		}
	}

	// 如果有命令行参数 --json，则输出 JSON 格式
	if len(os.Args) > 1 && strings.Contains(os.Args[1], "--json") {
		jsonOutput, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling to JSON: %v", err)
		}
		fmt.Println("\n======================= JSON输出 =======================")
		fmt.Println(string(jsonOutput))
	}
}

// formatSystemInfo 将系统信息格式化为指定的输出格式
func formatSystemInfo(info model.SystemInfo) string {
	var sb strings.Builder

	sb.WriteString("静态信息：\n")

	// 1. 计算机名和系统类型
	osType := "Mac"
	if runtime.GOOS == "windows" {
		osType = "Windows"
	}
	sb.WriteString(fmt.Sprintf("1. 计算机名（系统）：%s（%s）\n", info.Hostname, osType))

	// 2. 型号名称
	if info.Model != "" {
		sb.WriteString(fmt.Sprintf("2. 型号名称：%s\n", info.Model))
	}

	// 3. 型号标识符
	if info.ModelID != "" {
		sb.WriteString(fmt.Sprintf("3. 型号标识符：%s\n", info.ModelID))
	} else if info.Model != "" {
		// 如果没有ModelID但有Model，则使用Model作为标识符
		sb.WriteString(fmt.Sprintf("3. 型号标识符：%s\n", info.Model))
	} else {
		sb.WriteString("3. 型号标识符：未知\n")
	}

	// 4. 序列号
	sb.WriteString(fmt.Sprintf("4. SN： %s\n", info.SerialNumber))

	// 5. 处理器信息（包括核心数）
	cpuDesc := fmt.Sprintf("%s", info.CPU.Model)
	if info.CPU.Cores > 0 {
		// 根据CPU类型调整显示格式
		if strings.Contains(strings.ToLower(cpuDesc), "intel") {
			// Intel处理器格式：X核Intel Core iX
			cpuDesc = fmt.Sprintf("%d核%s", info.CPU.Cores, cpuDesc)
		} else if strings.Contains(cpuDesc, "Apple") {
			// Apple Silicon处理器格式：
			cpuDesc = fmt.Sprintf("%s (%d核)", cpuDesc, info.CPU.Cores)
		} else {
			// 其他处理器格式
			cpuDesc = fmt.Sprintf("%s (%d核)", cpuDesc, info.CPU.Cores)
		}
	}
	sb.WriteString(fmt.Sprintf("5. 处理器名称：%s\n", cpuDesc))

	// 6. 硬件UUID
	sb.WriteString(fmt.Sprintf("6. 硬件UUID: %s\n", info.UUID))

	// 7. 磁盘信息
	if len(info.Disks) > 0 {
		disk := info.Disks[0]
		diskDesc := disk.Model
		if diskDesc == "" {
			// 如果没有模型信息，则使用磁盘名称
			diskDesc = disk.Name
		}
		sb.WriteString(fmt.Sprintf("7. 磁盘: %s\n", diskDesc))
	} else {
		sb.WriteString("7. 磁盘: 未知\n")
	}

	// 8. CPU简短描述（仅型号）
	sb.WriteString(fmt.Sprintf("8. CPU: %s\n", info.CPU.Model))

	return sb.String()
}
