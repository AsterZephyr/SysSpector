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
	fmt.Println("======================= 系统信息 =======================")
	fmt.Printf("主机名: %s\n", info.Hostname)
	fmt.Printf("操作系统: %s\n", info.OS)
	fmt.Printf("系统版本: %s\n", info.SystemVersion)
	fmt.Printf("电脑名称: %s\n", info.ComputerName)
	fmt.Printf("启动时间: %s\n", info.UpTime)
	fmt.Printf("型号: %s\n", info.Model)
	if info.ModelID != "" {
		fmt.Printf("型号标识符: %s\n", info.ModelID)
	}
	fmt.Printf("序列号: %s\n", info.SerialNumber)
	fmt.Printf("UUID: %s\n", info.UUID)

	fmt.Println("\n======================= CPU信息 =======================")
	fmt.Printf("型号: %s\n", info.CPU.Model)
	fmt.Printf("核心数: %d\n", info.CPU.Cores)

	fmt.Println("\n======================= 内存信息 =======================")
	fmt.Printf("总容量: %.2f GB\n", float64(info.Memory.Total)/(1024*1024*1024))
	fmt.Printf("类型: %s\n", info.Memory.Type)

	// 显示内存使用情况
	if info.MemoryUsage.Total > 0 {
		fmt.Println("\n======================= 内存使用情况 =======================")
		fmt.Printf("总容量: %.2f GB\n", float64(info.MemoryUsage.Total)/(1024*1024*1024))
		fmt.Printf("已使用: %.2f GB (%.1f%%)\n", 
			float64(info.MemoryUsage.Used)/(1024*1024*1024), 
			float64(info.MemoryUsage.Used)/float64(info.MemoryUsage.Total)*100)
		fmt.Printf("空闲: %.2f GB\n", float64(info.MemoryUsage.Free)/(1024*1024*1024))
		fmt.Printf("活跃: %.2f GB\n", float64(info.MemoryUsage.Active)/(1024*1024*1024))
		fmt.Printf("不活跃: %.2f GB\n", float64(info.MemoryUsage.Inactive)/(1024*1024*1024))
		fmt.Printf("已缓存: %.2f GB\n", float64(info.MemoryUsage.Cached)/(1024*1024*1024))
	}

	fmt.Println("\n======================= 磁盘信息 =======================")
	for i, disk := range info.Disks {
		fmt.Printf("磁盘 %d:\n", i+1)
		fmt.Printf("  名称: %s\n", disk.Name)
		fmt.Printf("  型号: %s\n", disk.Model)
		fmt.Printf("  容量: %d GB\n", disk.Size)
		if disk.Serial != "" {
			fmt.Printf("  序列号: %s\n", disk.Serial)
		}
	}

	// 显示磁盘使用情况
	if len(info.DiskUsage) > 0 {
		fmt.Println("\n======================= 磁盘使用情况 =======================")
		for _, usage := range info.DiskUsage {
			fmt.Printf("挂载点: %s\n", usage.MountPoint)
			fmt.Printf("  文件系统: %s\n", usage.Filesystem)
			fmt.Printf("  总容量: %.2f GB\n", float64(usage.Total)/(1024*1024*1024))
			fmt.Printf("  已使用: %.2f GB (%.1f%%)\n", 
				float64(usage.Used)/(1024*1024*1024), 
				float64(usage.Used)/float64(usage.Total)*100)
			fmt.Printf("  可用: %.2f GB\n", float64(usage.Free)/(1024*1024*1024))
			fmt.Println()
		}
	}

	// 显示电池信息
	if info.Battery.IsPresent {
		fmt.Println("======================= 电池信息 =======================")
		fmt.Printf("电量: %.1f%%\n", info.Battery.Percentage)
		fmt.Printf("状态: %s\n", info.Battery.Status)
		fmt.Printf("健康度: %.1f%%\n", info.Battery.Health)
		fmt.Printf("循环次数: %d\n", info.Battery.CycleCount)
		if info.Battery.TimeRemaining > 0 {
			hours := info.Battery.TimeRemaining / 60
			minutes := info.Battery.TimeRemaining % 60
			fmt.Printf("剩余时间: %d小时%d分钟\n", hours, minutes)
		}
	}

	// 显示交流充电器信息
	if info.ACAdapter.IsConnected {
		fmt.Println("\n======================= 充电器信息 =======================")
		fmt.Printf("连接状态: %s\n", "已连接")
		fmt.Printf("瓦数: %d W\n", info.ACAdapter.Wattage)
	}

	// 显示蓝牙信息
	if info.Bluetooth.IsAvailable {
		fmt.Println("\n======================= 蓝牙信息 =======================")
		fmt.Printf("状态: %s\n", info.Bluetooth.Status)
		fmt.Printf("名称: %s\n", info.Bluetooth.Name)
		fmt.Printf("地址: %s\n", info.Bluetooth.Address)
		
		if len(info.Bluetooth.ConnectedDevices) > 0 {
			fmt.Println("已连接设备:")
			for _, device := range info.Bluetooth.ConnectedDevices {
				fmt.Printf("  - %s (%s)\n", device.Name, device.Type)
			}
		}
	}

	// 显示温度信息
	if len(info.Temperature) > 0 {
		fmt.Println("\n======================= 温度信息 =======================")
		for _, temp := range info.Temperature {
			fmt.Printf("%s: %.1f°C\n", temp.Sensor, temp.Value)
		}
	}

	// 显示WiFi自动连接状态
	if info.WiFiAutoJoin.IsConfigured {
		fmt.Println("\n======================= WiFi自动连接 =======================")
		fmt.Printf("状态: %s\n", info.WiFiAutoJoin.Status)
		if len(info.WiFiAutoJoin.Networks) > 0 {
			fmt.Println("已保存网络:")
			for _, network := range info.WiFiAutoJoin.Networks {
				fmt.Printf("  - %s (自动连接: %v)\n", network.SSID, network.AutoJoin)
			}
		}
	}

	// 显示网络信息
	if info.Network.WiFi.IsConnected {
		fmt.Println("\n======================= WiFi信息 =======================")
		fmt.Printf("SSID: %s\n", info.Network.WiFi.SSID)
		fmt.Printf("BSSID: %s\n", info.Network.WiFi.BSSID)
		fmt.Printf("信号强度: %d dBm\n", info.Network.WiFi.SignalStrength)
		fmt.Printf("频道: %d\n", info.Network.WiFi.Channel)
		fmt.Printf("频率: %.1f GHz\n", info.Network.WiFi.Frequency)
		fmt.Printf("速率: %d Mbps\n", info.Network.WiFi.TxRate)
	}

	if len(info.Network.Interfaces) > 0 {
		fmt.Println("\n======================= 网络接口 =======================")
		for _, iface := range info.Network.Interfaces {
			fmt.Printf("接口: %s\n", iface.Name)
			fmt.Printf("  MAC地址: %s\n", iface.MACAddress)
			fmt.Printf("  状态: %s\n", iface.Status)
			
			if len(iface.IPAddresses) > 0 {
				fmt.Println("  IP地址:")
				for _, ip := range iface.IPAddresses {
					fmt.Printf("    - %s\n", ip)
				}
			}
		}
	}

	if info.Network.PublicIP != "" {
		fmt.Println("\n======================= 公网IP =======================")
		fmt.Printf("IP地址: %s\n", info.Network.PublicIP)
	}

	if len(info.Network.DNSServers) > 0 {
		fmt.Println("\n======================= DNS服务器 =======================")
		for i, dns := range info.Network.DNSServers {
			fmt.Printf("%d: %s\n", i+1, dns)
		}
	}

	// 显示已安装应用信息
	if len(info.InstalledApps) > 0 {
		fmt.Println("\n======================= 已安装应用 =======================")
		for i, app := range info.InstalledApps {
			if i >= 10 { // 只显示前10个应用
				fmt.Printf("... 还有 %d 个应用 ...\n", len(info.InstalledApps)-10)
				break
			}
			fmt.Printf("%s", app.Name)
			if app.Version != "" {
				fmt.Printf(" (版本: %s)", app.Version)
			}
			fmt.Println()
		}
	}

	// 显示正在运行的应用信息
	if len(info.RunningApps) > 0 {
		fmt.Println("\n======================= 正在运行的应用 =======================")
		// 按CPU使用率排序（简单实现，实际可以使用sort包）
		for i := 0; i < len(info.RunningApps) && i < 10; i++ {
			app := info.RunningApps[i]
			fmt.Printf("%s (PID: %d)\n", app.Name, app.PID)
			fmt.Printf("  CPU: %.1f%%\n", app.CPU)
			fmt.Printf("  内存: %.2f MB\n", float64(app.Memory)/(1024*1024))
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
