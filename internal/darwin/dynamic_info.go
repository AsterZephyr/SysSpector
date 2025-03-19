package darwin

import (
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"fmt"

	"github.com/AsterZephyr/SysSpector/pkg/model"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// GetDynamicSystemInfo 收集macOS系统的动态硬件信息
func GetDynamicSystemInfo(info *model.SystemInfo) error {
	var err error

	// 收集硬盘使用情况
	err = getDiskUsage(info)
	if err != nil {
		log.Printf("Error getting disk usage: %v", err)
	}

	// 收集内存使用情况
	err = getMemoryUsage(info)
	if err != nil {
		log.Printf("Error getting memory usage: %v", err)
	}

	// 收集电池信息
	err = getBatteryInfo(info)
	if err != nil {
		log.Printf("Error getting battery info: %v", err)
	}

	// 收集交流充电器信息
	err = getACAdapterInfo(info)
	if err != nil {
		log.Printf("Error getting AC adapter info: %v", err)
	}

	// 收集蓝牙信息
	err = getBluetoothInfo(info)
	if err != nil {
		log.Printf("Error getting bluetooth info: %v", err)
	}

	// 收集设备温度信息
	err = getTemperatureInfo(info)
	if err != nil {
		log.Printf("Error getting temperature info: %v", err)
	}

	// 收集WiFi自动连接状态
	err = getWiFiAutoJoinInfo(info)
	if err != nil {
		log.Printf("Error getting WiFi auto join info: %v", err)
	}

	return nil
}

// getDiskUsage 获取硬盘使用情况
func getDiskUsage(info *model.SystemInfo) error {
	// 使用gopsutil获取根目录的磁盘使用情况
	usage, err := disk.Usage("/")
	if err != nil {
		return err
	}

	// 创建一个分区信息切片
	partitions := []model.DiskPartitionInfo{
		{
			MountPoint: "/",
			Filesystem: "apfs",
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			UsedPerc:   usage.UsedPercent,
		},
	}

	info.DiskUsage = partitions
	return nil
}

// getMemoryUsage 获取内存使用情况
func getMemoryUsage(info *model.SystemInfo) error {
	// 使用gopsutil获取内存使用情况
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	info.MemoryUsage = model.MemoryUsageInfo{
		Total:    memInfo.Total,
		Used:     memInfo.Used,
		Free:     memInfo.Free,
		UsedPerc: memInfo.UsedPercent,
	}

	return nil
}

// getBatteryInfo 获取电池信息
func getBatteryInfo(info *model.SystemInfo) error {
	// 使用pmset命令获取电池信息
	output, err := runCommand("pmset", "-g", "batt")
	if err != nil {
		return err
	}

	// 解析电池百分比和充电状态
	batteryInfo := model.BatteryInfo{}

	// 检查是否存在电池
	batteryInfo.IsPresent = !strings.Contains(output, "No batteries available")

	if !batteryInfo.IsPresent {
		info.Battery = batteryInfo
		return nil
	}

	// 使用正则表达式匹配电池百分比
	percentRegex := regexp.MustCompile(`(\d+)%`)
	percentMatches := percentRegex.FindStringSubmatch(output)
	if len(percentMatches) > 1 {
		percentage, _ := strconv.Atoi(percentMatches[1])
		batteryInfo.Percentage = percentage
	}

	// 检查充电状态
	batteryInfo.IsCharging = strings.Contains(output, "charging") && !strings.Contains(output, "discharging")

	// 检查剩余时间
	timeRegex := regexp.MustCompile(`(\d+):(\d+)`)
	timeMatches := timeRegex.FindStringSubmatch(output)
	if len(timeMatches) > 2 {
		hours, _ := strconv.Atoi(timeMatches[1])
		minutes, _ := strconv.Atoi(timeMatches[2])
		batteryInfo.TimeRemaining = hours*60 + minutes
	}

	// 获取电池循环计数和健康状态
	cycleOutput, err := runCommand("system_profiler", "SPPowerDataType")
	if err == nil {
		// 获取循环计数
		cycleRegex := regexp.MustCompile(`Cycle Count: (\d+)`)
		cycleMatches := cycleRegex.FindStringSubmatch(cycleOutput)
		if len(cycleMatches) > 1 {
			cycleCount, _ := strconv.Atoi(cycleMatches[1])
			batteryInfo.CycleCount = cycleCount
		}

		// 获取电池健康状态
		healthRegex := regexp.MustCompile(`Condition: (.+)`)
		healthMatches := healthRegex.FindStringSubmatch(cycleOutput)
		if len(healthMatches) > 1 {
			batteryInfo.Health = strings.TrimSpace(healthMatches[1])
		}

		// 获取最大容量
		maxCapacityRegex := regexp.MustCompile(`Maximum Capacity: (\d+)%`)
		maxCapacityMatches := maxCapacityRegex.FindStringSubmatch(cycleOutput)
		if len(maxCapacityMatches) > 1 {
			maxCapacity, _ := strconv.Atoi(maxCapacityMatches[1])
			batteryInfo.Status = fmt.Sprintf("最大容量: %d%%", maxCapacity)
		}
	}

	info.Battery = batteryInfo
	return nil
}

// getACAdapterInfo 获取交流充电器信息
func getACAdapterInfo(info *model.SystemInfo) error {
	// 使用system_profiler获取电源信息，这与shell脚本一致
	output, err := runCommand("system_profiler", "SPPowerDataType")
	if err != nil {
		return err
	}

	// 解析交流充电器信息
	adapterInfo := model.ACAdapterInfo{}
	
	// 检查是否连接了交流充电器
	adapterInfo.Connected = strings.Contains(output, "AC Charger Information:")
	adapterInfo.IsConnected = adapterInfo.Connected // 设置兼容性字段
	
	if adapterInfo.Connected {
		// 尝试获取充电器序列号
		serialRegex := regexp.MustCompile(`Serial Number: (.+)`)
		serialMatches := serialRegex.FindStringSubmatch(output)
		if len(serialMatches) > 1 {
			adapterInfo.SerialNum = strings.TrimSpace(serialMatches[1])
		}
		
		// 尝试获取充电器名称
		nameRegex := regexp.MustCompile(`Name: (.+)`)
		nameMatches := nameRegex.FindStringSubmatch(output)
		if len(nameMatches) > 1 {
			adapterInfo.Name = strings.TrimSpace(nameMatches[1])
			
			// 尝试从名称中提取功率
			wattageRegex := regexp.MustCompile(`(\d+)W`)
			wattageMatches := wattageRegex.FindStringSubmatch(adapterInfo.Name)
			if len(wattageMatches) > 1 {
				wattage, _ := strconv.Atoi(wattageMatches[1])
				adapterInfo.Wattage = wattage
			}
		}
	}

	info.ACAdapter = adapterInfo
	return nil
}

// getBluetoothInfo 获取蓝牙信息
func getBluetoothInfo(info *model.SystemInfo) error {
	// 使用system_profiler获取蓝牙信息
	output, err := runCommand("system_profiler", "SPBluetoothDataType")
	if err != nil {
		return err
	}

	// 解析蓝牙状态
	bluetoothInfo := model.BluetoothInfo{}
	bluetoothInfo.Enabled = strings.Contains(output, "State: On") || strings.Contains(output, "Bluetooth: On")
	bluetoothInfo.Status = "关闭"
	if bluetoothInfo.Enabled {
		bluetoothInfo.Status = "打开"
	}

	// 解析已连接设备
	var connectedDevices []model.BTDeviceInfo

	// 使用正则表达式匹配连接的设备
	deviceRegex := regexp.MustCompile(`(?s)Connected: Yes.*?Address: ([0-9a-fA-F:]+).*?Name: ([^\n]+)`)
	deviceMatches := deviceRegex.FindAllStringSubmatch(output, -1)

	for _, match := range deviceMatches {
		if len(match) > 2 {
			device := model.BTDeviceInfo{
				Address:   match[1],
				Name:      strings.TrimSpace(match[2]),
				Connected: true,
			}

			// 尝试确定设备类型
			deviceName := strings.ToLower(device.Name)
			if strings.Contains(deviceName, "keyboard") {
				device.Type = "键盘"
			} else if strings.Contains(deviceName, "mouse") || strings.Contains(deviceName, "trackpad") {
				device.Type = "鼠标/触控板"
			} else if strings.Contains(deviceName, "airpods") || strings.Contains(deviceName, "headphone") || strings.Contains(deviceName, "earphone") {
				device.Type = "耳机"
			} else if strings.Contains(deviceName, "speaker") {
				device.Type = "扬声器"
			} else {
				device.Type = "其他"
			}

			connectedDevices = append(connectedDevices, device)
		}
	}

	// 如果没有找到连接的设备，尝试使用另一种方式解析
	if len(connectedDevices) == 0 {
		// 尝试提取设备名称
		deviceNameRegex := regexp.MustCompile(`Connected:\s+Yes[\s\S]*?^\s*([^:]+):\s*$`)
		deviceNameMatches := deviceNameRegex.FindAllStringSubmatch(output, -1)

		for _, match := range deviceNameMatches {
			if len(match) > 1 {
				deviceName := strings.TrimSpace(match[1])
				if deviceName != "" {
					device := model.BTDeviceInfo{
						Name:      deviceName,
						Connected: true,
					}

					// 尝试确定设备类型
					lowerName := strings.ToLower(deviceName)
					if strings.Contains(lowerName, "keyboard") {
						device.Type = "键盘"
					} else if strings.Contains(lowerName, "mouse") || strings.Contains(lowerName, "trackpad") {
						device.Type = "鼠标/触控板"
					} else if strings.Contains(lowerName, "airpods") || strings.Contains(lowerName, "headphone") || strings.Contains(lowerName, "earphone") {
						device.Type = "耳机"
					} else if strings.Contains(lowerName, "speaker") {
						device.Type = "扬声器"
					} else {
						device.Type = "其他"
					}

					connectedDevices = append(connectedDevices, device)
				}
			}
		}
	}

	bluetoothInfo.Devices = connectedDevices
	info.Bluetooth = bluetoothInfo
	return nil
}

// getTemperatureInfo 获取设备温度信息
func getTemperatureInfo(info *model.SystemInfo) error {
	// 使用sysctl命令获取温度信息
	cmd := exec.Command("sysctl", "-a")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("获取温度信息失败: %v", err)
	}

	outputStr := string(output)

	// 查找CPU温度
	cpuTempRegex := regexp.MustCompile(`machdep.xcpm.cpu_thermal_level:\s+(\d+)`)
	cpuTempMatches := cpuTempRegex.FindStringSubmatch(outputStr)
	var cpuTemp float64
	if len(cpuTempMatches) > 1 {
		cpuTemp, _ = strconv.ParseFloat(cpuTempMatches[1], 64)
		cpuTemp *= 10 // 转换为摄氏度
	}

	// 查找GPU温度
	gpuTempRegex := regexp.MustCompile(`hw.gpufrequency.thermal_level:\s+(\d+)`)
	gpuTempMatches := gpuTempRegex.FindStringSubmatch(outputStr)
	var gpuTemp float64
	if len(gpuTempMatches) > 1 {
		gpuTemp, _ = strconv.ParseFloat(gpuTempMatches[1], 64)
	}

	// 创建一个温度传感器信息切片
	sensors := []model.TempSensorInfo{
		{
			Name:        "CPU",
			Temperature: cpuTemp,
			Location:    "处理器",
			Sensor:      "CPU",
			Value:       cpuTemp,
		},
		{
			Name:        "GPU",
			Temperature: gpuTemp,
			Location:    "图形处理器",
			Sensor:      "GPU",
			Value:       gpuTemp,
		},
	}

	info.Temperature = sensors
	return nil
}

// getWiFiAutoJoinInfo 获取WiFi自动连接状态
func getWiFiAutoJoinInfo(info *model.SystemInfo) error {
	// 检查WiFi网络配置文件
	plistPath := "/Library/Preferences/com.apple.network.plist"

	// 检查文件是否存在
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		// 文件不存在，无法获取自动连接状态
		info.WiFiAutoJoin = model.WiFiAutoJoinInfo{
			IsConfigured: false,
			Status:       "未配置",
			Networks:     []model.WiFiNetworkInfo{},
		}
		return nil
	}

	// 获取当前连接的WiFi网络SSID
	currentSSID := ""
	if info.Network.WiFi.IsConnected {
		currentSSID = info.Network.WiFi.SSID
	}

	// 如果没有连接WiFi，则返回默认状态
	if currentSSID == "" {
		info.WiFiAutoJoin = model.WiFiAutoJoinInfo{
			IsConfigured: true,
			Status:       "已配置",
			Networks:     []model.WiFiNetworkInfo{},
		}
		return nil
	}

	// 查找当前网络是否配置了自动连接
	autoJoin := true // 默认为自动连接

	// 创建WiFi自动连接信息
	info.WiFiAutoJoin = model.WiFiAutoJoinInfo{
		IsConfigured: true,
		Status:       "已配置",
		Networks: []model.WiFiNetworkInfo{
			{
				SSID:     currentSSID,
				AutoJoin: autoJoin,
			},
		},
	}

	return nil
}
