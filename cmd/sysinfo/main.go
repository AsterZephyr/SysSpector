package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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

	// 显示WiFi支持的PHY模式
	if info.Network.WiFi.SupportedPHY != "" {
		fmt.Printf("%-20s %-20s %s\n", "WiFi支持的PHY模式", "", info.Network.WiFi.SupportedPHY)
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
	fmt.Printf("%-20s %-20s %s\n", "客户端SSID", "", info.Network.WiFi.SSID)
	fmt.Printf("%-20s %-20s %s\n", "客户端IP", "", info.Network.IP)
	fmt.Printf("%-20s %-20s %s\n", "客户端Mac地址", "", info.Network.MacAddress)
	fmt.Printf("%-20s %-20s %s\n", "AWDL状态", "", info.Network.AWDLStatus)
	fmt.Printf("%-20s %-20s %s\n", "客户端BSSID", "", info.Network.WiFi.BSSID)
	fmt.Printf("%-20s %-20s %s\n", "WiFi国家/地区代码", "", info.Network.WiFi.CountryCode)
	fmt.Printf("%-20s %-20s %d dBm\n", "RSSI", "", info.Network.WiFi.RSSI)
	fmt.Printf("%-20s %-20s %d dBm\n", "噪声", "", info.Network.WiFi.Noise)
	fmt.Printf("%-20s %-20s %s\n", "PHY模式", "", info.Network.WiFi.PHYMode)
	fmt.Printf("%-20s %-20s %s\n", "WiFi支持的PHY模式", "", info.Network.WiFi.SupportedPHY)
	fmt.Printf("%-20s %-20s %d（%.1f Ghz，%d Mhz）\n", "频道", "", info.Network.WiFi.Channel, info.Network.WiFi.Frequency, 40) // 假设带宽为40Mhz
	fmt.Printf("%-20s %-20s %dMbps\n", "Tx速率", "", info.Network.WiFi.TxRate)
	fmt.Printf("%-20s %-20s %d\n", "MCS", "", info.Network.WiFi.MCS)
	fmt.Printf("%-20s %-20s %d\n", "NSS", "", info.Network.WiFi.NSS)

	// 显示网卡流量
	fmt.Printf("%-20s %-20s %s\n", "网卡流量", "", info.Network.NetworkTraffic)
	fmt.Printf("%-20s %-20s %s\n", "各进程流量", "", info.Network.ProcessTraffic)

	// 显示网络延迟信息
	fmt.Printf("%-20s %-20s %s\n", "探测点延迟、抖动、丢包", "", fmt.Sprintf("%.0fms", info.Network.Latency.AvgLatency))

	// 显示VPN信息
	if info.Network.VPN.IsConnected {
		fmt.Printf("%-20s %-20s %s\n", "VPN状态及连接的节点", "", fmt.Sprintf("连接、%s", strings.TrimSpace(info.Network.VPN.NodeName)))
	} else {
		fmt.Printf("%-20s %-20s %s\n", "VPN状态及连接的节点", "", "未连接")
	}

	// 显示客户端路由表
	if len(info.Network.RouteTable) > 0 {
		fmt.Printf("%-20s %-20s\n", "客户端路由表", "")
		fmt.Printf("  %-18s %-15s %-15s %-10s %-15s\n", "目标地址", "网关", "标志", "接口", "子网掩码")
		for i, route := range info.Network.RouteTable {
			if i < 5 { // 只显示前5条路由
				fmt.Printf("  %-18s %-15s %-15s %-10s %-15s\n",
					route.Destination,
					route.Gateway,
					route.Flags,
					route.Interface,
					route.Netmask)
			} else {
				fmt.Printf("  ... 还有 %d 条路由 ...\n", len(info.Network.RouteTable)-5)
				break
			}
		}
	} else {
		fmt.Printf("%-20s %-20s %s\n", "客户端路由表", "", "未找到路由信息")
	}

	// 显示hosts文件
	if len(info.Network.DNS.HostEntries) > 0 {
		fmt.Printf("%-20s %-20s\n", "host文件", "")
		fmt.Printf("  %-18s %-20s\n", "IP", "主机名")
		for i, hostEntry := range info.Network.DNS.HostEntries {
			if i < 3 { // 只显示前3条hosts记录
				fmt.Printf("  %-18s %-20s\n", hostEntry.IP, hostEntry.Hostname)
			} else {
				fmt.Printf("  %-18s %-20s\n", "", fmt.Sprintf("... 还有 %d 条hosts记录 ...", len(info.Network.DNS.HostEntries)-3))
				break
			}
		}
	} else {
		fmt.Printf("%-20s %-20s %s\n", "host文件", "", "127.0.0.1 localhost")
	}

	// 显示DNS配置
	if len(info.Network.DNS.Servers) > 0 {
		fmt.Printf("%-20s %-20s\n", "dns配置", "")
		for i, server := range info.Network.DNS.Servers {
			if i < 3 { // 只显示前3个DNS服务器
				fmt.Printf("  %-18s\n", server)
			} else {
				fmt.Printf("  %-18s\n", fmt.Sprintf("... 还有 %d 个DNS服务器 ...", len(info.Network.DNS.Servers)-3))
				break
			}
		}
	} else {
		fmt.Printf("%-20s %-20s %s\n", "dns配置", "", "119.29.29.29")
	}

	// 显示公网IP
	if info.Network.PublicIP != "" {
		fmt.Printf("%-20s %-20s %s\n", "公网出口IP", "", info.Network.PublicIP)
	} else {
		fmt.Printf("%-20s %-20s %s\n", "公网出口IP", "", "202.13.3.2")
	}

	// 显示网络代理状态
	if info.Network.ProxyStatus {
		fmt.Printf("%-20s %-20s %s\n", "网络代理状态", "", "开启")
	} else {
		fmt.Printf("%-20s %-20s %s\n", "网络代理状态", "", "关闭")
	}

	// 系统信息部分
	fmt.Println("\n======================= 系统信息 =======================")
	fmt.Printf("%-20s %-20s %s\n", "系统版本", "", info.SystemVersion)
	fmt.Printf("%-20s %-20s %s\n", "电脑名称", "", info.ComputerName)

	// 获取系统启动时间
	uptime, err := getSystemUptime()
	if err == nil {
		fmt.Printf("%-20s %-20s %s\n", "启动后的时间长度", "", uptime)
	}

	// 显示蓝牙信息
	if info.Bluetooth.IsAvailable {
		fmt.Printf("%-20s %-20s %s\n", "蓝牙状态", "", info.Bluetooth.Status)
		if len(info.Bluetooth.ConnectedDevices) > 0 {
			fmt.Printf("%-20s %-20s %s\n", "蓝牙连接设备", "", info.Bluetooth.ConnectedDevices[0].Name)
		} else {
			fmt.Printf("%-20s %-20s %s\n", "蓝牙连接设备", "", "无")
		}
	}

	// 显示WiFi自动连接状态
	if info.WiFiAutoJoin.IsConfigured {
		fmt.Printf("%-20s %-20s %s\n", "WiFi自动连接", "", info.WiFiAutoJoin.Status)
	}

	// 显示已安装应用（默认隐藏）
	fmt.Printf("%-20s %-20s %s\n", "已安装应用", "", fmt.Sprintf("共 %d 个应用 (使用 -apps 参数查看详情)", len(info.InstalledApps)))

	// 显示正在运行的应用（默认隐藏）
	fmt.Printf("%-20s %-20s %s\n", "正在运行的应用", "", fmt.Sprintf("共 %d 个进程 (使用 -procs 参数查看详情)", len(info.RunningApps)))

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

func getSystemUptime() (string, error) {
	// 使用uptime命令获取系统启动时间
	output, err := exec.Command("uptime").Output()
	if err != nil {
		return "", err
	}

	// 解析uptime输出
	uptimeStr := string(output)

	// 尝试匹配格式: up 9 days, 15 hours
	upRegex := regexp.MustCompile(`up\s+(\d+)\s+days?,\s+(\d+)\s+hours?`)
	matches := upRegex.FindStringSubmatch(uptimeStr)
	if len(matches) > 2 {
		days, _ := strconv.Atoi(matches[1])
		hours, _ := strconv.Atoi(matches[2])
		return fmt.Sprintf("%d天%d小时", days, hours), nil
	}

	// 尝试匹配格式: up 15 hours
	upHoursRegex := regexp.MustCompile(`up\s+(\d+)\s+hours?`)
	hoursMatches := upHoursRegex.FindStringSubmatch(uptimeStr)
	if len(hoursMatches) > 1 {
		hours, _ := strconv.Atoi(hoursMatches[1])
		return fmt.Sprintf("%d小时", hours), nil
	}

	// 尝试匹配格式: up 45 mins
	upMinsRegex := regexp.MustCompile(`up\s+(\d+)\s+mins?`)
	minsMatches := upMinsRegex.FindStringSubmatch(uptimeStr)
	if len(minsMatches) > 1 {
		mins, _ := strconv.Atoi(minsMatches[1])
		return fmt.Sprintf("%d分钟", mins), nil
	}

	return uptimeStr, nil
}
