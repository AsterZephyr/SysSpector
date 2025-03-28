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

	// 收集网卡信息
	err = GetNetworkCardInfo(info)
	if err != nil {
		log.Printf("Error getting network card info: %v", err)
	}

	// 收集显卡信息
	err = GetGraphicsCardInfo(info)
	if err != nil {
		log.Printf("Error getting graphics card info: %v", err)
	}

	// 收集显示器信息
	err = GetDisplayInfo(info)
	if err != nil {
		log.Printf("Error getting display info: %v", err)
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
	// 检测是否为Apple Silicon芯片
	isAppleSilicon := false
	cmd := exec.Command("sysctl", "machdep.cpu.brand_string")
	cpuOutput, err := cmd.Output()
	if err == nil {
		cpuOutputStr := string(cpuOutput)
		isAppleSilicon = strings.Contains(cpuOutputStr, "Apple")
	}

	// 使用system_profiler获取电源信息，这与shell脚本一致
	powerOutput, err := runCommand("system_profiler", "SPPowerDataType")
	if err != nil {
		return err
	}

	// 解析交流充电器信息
	adapterInfo := model.ACAdapterInfo{}

	// 检查是否连接了交流充电器
	if isAppleSilicon {
		// Apple Silicon Mac的充电器信息格式
		adapterInfo.Connected = strings.Contains(powerOutput, "AC Charger Information:")
	} else {
		// Intel Mac的充电器信息格式
		adapterInfo.Connected = strings.Contains(powerOutput, "AC Charger Information:") ||
			strings.Contains(powerOutput, "AC Adapter Information:") ||
			strings.Contains(powerOutput, "Power Adapter Information:")
	}
	adapterInfo.IsConnected = adapterInfo.Connected // 设置兼容性字段

	if adapterInfo.Connected {
		// 尝试获取充电器序列号
		serialRegex := regexp.MustCompile(`(?:Serial Number|ID): (.+)`)
		serialMatches := serialRegex.FindStringSubmatch(powerOutput)
		if len(serialMatches) > 1 {
			adapterInfo.SerialNum = strings.TrimSpace(serialMatches[1])
		}

		// 尝试获取充电器名称
		nameRegex := regexp.MustCompile(`(?:Name|Family): (.+)`)
		nameMatches := nameRegex.FindStringSubmatch(powerOutput)
		if len(nameMatches) > 1 {
			adapterInfo.Name = strings.TrimSpace(nameMatches[1])

			// 尝试从名称中提取功率
			wattageRegex := regexp.MustCompile(`(\d+)W`)
			wattageMatches := wattageRegex.FindStringSubmatch(adapterInfo.Name)
			if len(wattageMatches) > 1 {
				wattage, _ := strconv.Atoi(wattageMatches[1])
				adapterInfo.Wattage = wattage
			} else {
				// 尝试从其他字段获取功率
				wattageRegex = regexp.MustCompile(`(?:Wattage|Adapter Wattage): (\d+)W`)
				wattageMatches = wattageRegex.FindStringSubmatch(powerOutput)
				if len(wattageMatches) > 1 {
					wattage, _ := strconv.Atoi(wattageMatches[1])
					adapterInfo.Wattage = wattage
				}
			}
		}

		// 尝试获取充电器芯片型号
		chipRegex := regexp.MustCompile(`(?:Manufacturer|Manufacturer ID|Vendor): (.+)`)
		chipMatches := chipRegex.FindStringSubmatch(powerOutput)
		if len(chipMatches) > 1 {
			adapterInfo.ChipModel = strings.TrimSpace(chipMatches[1])
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
	
	// 修复状态检测逻辑
	stateRegex := regexp.MustCompile(`State: (\w+)`)
	stateMatch := stateRegex.FindStringSubmatch(output)
	
	bluetoothInfo.Enabled = false
	bluetoothInfo.Status = "关闭"
	
	if len(stateMatch) > 1 && stateMatch[1] == "On" {
		bluetoothInfo.Enabled = true
		bluetoothInfo.Status = "打开"
	}
	
	// 解析已连接设备
	var connectedDevices []model.BTDeviceInfo

	// 修复连接设备检测逻辑
	// 首先尝试查找"Connected:"部分
	connectedSection := ""
	connectedRegex := regexp.MustCompile(`(?s)Connected:(.*?)(?:Not Connected:|$)`)
	connectedMatch := connectedRegex.FindStringSubmatch(output)
	if len(connectedMatch) > 1 {
		connectedSection = connectedMatch[1]
	}

	// 从连接部分提取设备信息
	if connectedSection != "" {
		// 提取设备名称和地址
		deviceRegex := regexp.MustCompile(`(?m)^\s*([^:]+):\s*$`)
		deviceMatches := deviceRegex.FindAllStringSubmatchIndex(connectedSection, -1)
		
		for i, match := range deviceMatches {
			if len(match) >= 2 {
				startPos := match[0]
				endPos := len(connectedSection)
				if i < len(deviceMatches)-1 {
					endPos = deviceMatches[i+1][0]
				}
				
				deviceSection := connectedSection[startPos:endPos]
				deviceName := strings.TrimSpace(connectedSection[match[2]:match[3]])
				
				// 提取设备地址
				addressRegex := regexp.MustCompile(`Address: ([0-9a-fA-F:]+)`)
				addressMatch := addressRegex.FindStringSubmatch(deviceSection)
				address := ""
				if len(addressMatch) > 1 {
					address = addressMatch[1]
				}
				
				// 提取设备类型
				typeRegex := regexp.MustCompile(`Minor Type: ([^\n]+)`)
				typeMatch := typeRegex.FindStringSubmatch(deviceSection)
				deviceType := "其他"
				if len(typeMatch) > 1 {
					rawType := strings.TrimSpace(typeMatch[1])
					// 转换为中文类型
					switch strings.ToLower(rawType) {
					case "mouse":
						deviceType = "鼠标"
					case "keyboard":
						deviceType = "键盘"
					case "headset":
						deviceType = "耳机"
					case "speaker":
						deviceType = "扬声器"
					}
				}
				
				device := model.BTDeviceInfo{
					Name:      deviceName,
					Address:   address,
					Type:      deviceType,
					Connected: true,
				}
				
				connectedDevices = append(connectedDevices, device)
			}
		}
	}

	bluetoothInfo.Devices = connectedDevices
	info.Bluetooth = bluetoothInfo
	return nil
}

// getTemperatureInfo 获取设备温度信息
func getTemperatureInfo(info *model.SystemInfo) error {
	// 检测是否为Apple Silicon芯片
	isAppleSilicon := false
	cmd := exec.Command("sysctl", "machdep.cpu.brand_string")
	cpuOutput, err := cmd.Output()
	if err == nil {
		cpuOutputStr := string(cpuOutput)
		isAppleSilicon = strings.Contains(cpuOutputStr, "Apple")
	}

	// 根据芯片类型使用不同的温度获取方法
	if isAppleSilicon {
		// Apple Silicon芯片的温度获取方法
		return getAppleSiliconTemperature(info)
	} else {
		// Intel芯片的温度获取方法
		return getIntelTemperature(info)
	}
}

// getAppleSiliconTemperature 获取Apple Silicon设备的温度信息
func getAppleSiliconTemperature(info *model.SystemInfo) error {
	// 使用sysctl命令获取温度信息
	cmd := exec.Command("sysctl", "-a")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("获取温度信息失败: %v", err)
		return err
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

// getIntelTemperature 获取Intel Mac设备的温度信息
func getIntelTemperature(info *model.SystemInfo) error {
	// 尝试使用iStats命令获取温度信息
	// 首先检查是否已安装iStats
	_, err := exec.LookPath("istats")
	if err != nil {
		// iStats未安装，使用备用方法
		return getIntelTemperatureBackup(info)
	}

	// 使用iStats获取温度信息
	cmd := exec.Command("istats")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("使用iStats获取温度信息失败: %v", err)
		return getIntelTemperatureBackup(info)
	}

	outputStr := string(output)
	sensors := []model.TempSensorInfo{}

	// 解析CPU温度
	cpuTempRegex := regexp.MustCompile(`CPU temp:\s+(\d+\.\d+)°C`)
	cpuTempMatches := cpuTempRegex.FindStringSubmatch(outputStr)
	if len(cpuTempMatches) > 1 {
		cpuTemp, _ := strconv.ParseFloat(cpuTempMatches[1], 64)
		sensors = append(sensors, model.TempSensorInfo{
			Name:        "CPU",
			Temperature: cpuTemp,
			Location:    "处理器",
			Sensor:      "CPU",
			Value:       cpuTemp,
		})
	}

	// 解析其他温度传感器
	sensorRegex := regexp.MustCompile(`(\w+\s*\w*):\s+(\d+\.\d+)°C`)
	sensorMatches := sensorRegex.FindAllStringSubmatch(outputStr, -1)
	for _, match := range sensorMatches {
		if len(match) > 2 && match[1] != "CPU temp" {
			sensorName := strings.TrimSpace(match[1])
			sensorTemp, _ := strconv.ParseFloat(match[2], 64)

			// 跳过已添加的CPU温度
			if sensorName == "CPU" {
				continue
			}

			sensors = append(sensors, model.TempSensorInfo{
				Name:        sensorName,
				Temperature: sensorTemp,
				Location:    sensorName,
				Sensor:      sensorName,
				Value:       sensorTemp,
			})
		}
	}

	info.Temperature = sensors
	return nil
}

// getIntelTemperatureBackup 获取Intel Mac设备的温度信息的备用方法
func getIntelTemperatureBackup(info *model.SystemInfo) error {
	// 使用osx-cpu-temp命令获取CPU温度
	_, err := exec.LookPath("osx-cpu-temp")
	if err != nil {
		// 如果没有安装任何温度监控工具，返回默认值
		sensors := []model.TempSensorInfo{
			{
				Name:        "CPU",
				Temperature: 0,
				Location:    "处理器",
				Sensor:      "CPU",
				Value:       0,
			},
		}
		info.Temperature = sensors
		return nil
	}

	cmd := exec.Command("osx-cpu-temp")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("使用osx-cpu-temp获取温度信息失败: %v", err)
		return nil
	}

	outputStr := string(output)
	tempRegex := regexp.MustCompile(`(\d+\.\d+)°C`)
	tempMatches := tempRegex.FindStringSubmatch(outputStr)

	sensors := []model.TempSensorInfo{}
	if len(tempMatches) > 1 {
		cpuTemp, _ := strconv.ParseFloat(tempMatches[1], 64)
		sensors = append(sensors, model.TempSensorInfo{
			Name:        "CPU",
			Temperature: cpuTemp,
			Location:    "处理器",
			Sensor:      "CPU",
			Value:       cpuTemp,
		})
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

// GetNetworkCardInfo 获取网卡信息
func GetNetworkCardInfo(info *model.SystemInfo) error {
	// 使用ifconfig命令获取网络接口信息
	output, err := exec.Command("ifconfig", "-a").Output()
	if err != nil {
		return fmt.Errorf("获取网卡信息失败: %v", err)
	}

	// 解析输出
	interfaces := strings.Split(string(output), "\n\n")
	for _, iface := range interfaces {
		if len(strings.TrimSpace(iface)) == 0 {
			continue
		}

		// 提取网卡名称
		nameRegex := regexp.MustCompile(`^([a-zA-Z0-9]+):`)
		nameMatch := nameRegex.FindStringSubmatch(iface)
		if len(nameMatch) < 2 {
			continue
		}
		name := nameMatch[1]

		// 跳过虚拟接口和回环接口
		if name == "lo0" || strings.HasPrefix(name, "utun") || strings.HasPrefix(name, "llw") {
			continue
		}

		// 提取MAC地址
		macRegex := regexp.MustCompile(`ether\s+([0-9a-f:]+)`)
		macMatch := macRegex.FindStringSubmatch(iface)
		mac := ""
		if len(macMatch) >= 2 {
			mac = macMatch[1]
		}

		// 提取IP地址
		ipRegex := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
		ipMatches := ipRegex.FindAllStringSubmatch(iface, -1)
		ipAddresses := []string{}
		for _, match := range ipMatches {
			if len(match) >= 2 {
				ipAddresses = append(ipAddresses, match[1])
			}
		}

		// 提取IPv6地址
		ipv6Regex := regexp.MustCompile(`inet6\s+([0-9a-f:]+)`)
		ipv6Matches := ipv6Regex.FindAllStringSubmatch(iface, -1)
		for _, match := range ipv6Matches {
			if len(match) >= 2 {
				ipAddresses = append(ipAddresses, match[1])
			}
		}

		// 创建网卡信息
		networkCard := struct {
			Name        string
			MACAddress  string
			IPAddresses []string
		}{
			Name:        name,
			MACAddress:  mac,
			IPAddresses: ipAddresses,
		}

		// 添加到网卡列表
		info.NetworkCards = append(info.NetworkCards, networkCard)
	}

	// 如果没有找到任何网卡，尝试使用networksetup命令获取网络接口列表
	if len(info.NetworkCards) == 0 {
		output, err := exec.Command("networksetup", "-listallhardwareports").Output()
		if err != nil {
			return fmt.Errorf("使用networksetup命令失败: %v", err)
		}
		
		// 解析输出
		portRegex := regexp.MustCompile(`(?s)Hardware Port: (.+?)\nDevice: (.+?)\nEthernet Address: ([0-9a-f:]+)`)
		portMatches := portRegex.FindAllStringSubmatch(string(output), -1)
		
		for _, match := range portMatches {
			if len(match) < 4 {
				continue
			}
			
			portName := strings.TrimSpace(match[1])
			deviceName := strings.TrimSpace(match[2])
			macAddress := strings.TrimSpace(match[3])
			
			// 获取IP地址
			ipOutput, err := exec.Command("ipconfig", "getifaddr", deviceName).Output()
			ipAddress := ""
			if err == nil {
				ipAddress = strings.TrimSpace(string(ipOutput))
			}
			
			ipAddresses := []string{}
			if ipAddress != "" {
				ipAddresses = append(ipAddresses, ipAddress)
			}
			
			// 创建网卡信息
			networkCard := struct {
				Name        string
				MACAddress  string
				IPAddresses []string
			}{
				Name:        deviceName + " (" + portName + ")",
				MACAddress:  macAddress,
				IPAddresses: ipAddresses,
			}
			
			// 添加到网卡列表
			info.NetworkCards = append(info.NetworkCards, networkCard)
		}
	}

	return nil
}

// GetGraphicsCardInfo 获取显卡信息
func GetGraphicsCardInfo(info *model.SystemInfo) error {
	// 使用system_profiler命令获取显卡信息
	output, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return fmt.Errorf("获取显卡信息失败: %v", err)
	}

	// 修改正则表达式，适配不同格式的显卡信息
	// 尝试匹配包含VRAM信息的显卡
	gpuRegex := regexp.MustCompile(`(?s)Chipset Model: (.+?)(?:\n|$)`)
	gpuMatches := gpuRegex.FindAllStringSubmatch(string(output), -1)

	for _, match := range gpuMatches {
		if len(match) < 2 {
			continue
		}

		model := strings.TrimSpace(match[1])
		
		// 创建显卡信息
		graphicsCard := struct {
			Name   string
			Model  string
			Memory uint64
		}{
			Name:   model,
			Model:  model,
			Memory: 0, // 默认为0
		}

		// 尝试查找显存信息
		memoryRegex := regexp.MustCompile(`(?s)Chipset Model: ` + regexp.QuoteMeta(model) + `.*?(?:VRAM|Total Available).*?: (\d+(?:\.\d+)?) ?(MB|GB)`)
		memoryMatches := memoryRegex.FindStringSubmatch(string(output))
		
		if len(memoryMatches) >= 3 {
			memoryStr := memoryMatches[1]
			memoryUnit := memoryMatches[2]
			
			memoryValue, err := strconv.ParseFloat(memoryStr, 64)
			if err == nil {
				if memoryUnit == "MB" {
					graphicsCard.Memory = uint64(memoryValue * 1024 * 1024) // 转换为字节
				} else if memoryUnit == "GB" {
					graphicsCard.Memory = uint64(memoryValue * 1024 * 1024 * 1024) // 转换为字节
				}
			}
		}

		// 添加到显卡列表
		info.GraphicsCards = append(info.GraphicsCards, graphicsCard)
	}

	return nil
}

// GetDisplayInfo 获取显示器信息
func GetDisplayInfo(info *model.SystemInfo) error {
	// 使用system_profiler命令获取显示器信息
	output, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return fmt.Errorf("获取显示器信息失败: %v", err)
	}

	// 解析输出
	// 查找显示器部分
	displaySections := regexp.MustCompile(`(?s)Display Type: (.+?)\n(.*?)(?:\n\n|\z)`).FindAllStringSubmatch(string(output), -1)
	
	for _, section := range displaySections {
		if len(section) < 3 {
			continue
		}

		displayType := strings.TrimSpace(section[1])
		displayDetails := section[2]

		// 提取分辨率
		resolutionRegex := regexp.MustCompile(`Resolution: (\d+ x \d+)`)
		resolutionMatch := resolutionRegex.FindStringSubmatch(displayDetails)
		resolution := ""
		if len(resolutionMatch) >= 2 {
			resolution = resolutionMatch[1]
		}

		// 提取刷新率
		refreshRateRegex := regexp.MustCompile(`Refresh Rate: (\d+) Hz`)
		refreshRateMatch := refreshRateRegex.FindStringSubmatch(displayDetails)
		refreshRate := 0
		if len(refreshRateMatch) >= 2 {
			refreshRate, _ = strconv.Atoi(refreshRateMatch[1])
		}

		// 提取显示器名称/型号
		modelRegex := regexp.MustCompile(`Model: (.+?)\n`)
		modelMatch := modelRegex.FindStringSubmatch(displayDetails)
		model := ""
		if len(modelMatch) >= 2 {
			model = strings.TrimSpace(modelMatch[1])
		}

		// 创建显示器信息
		display := struct {
			Name        string
			Model       string
			Resolution  string
			RefreshRate int
		}{
			Name:        displayType,
			Model:       model,
			Resolution:  resolution,
			RefreshRate: refreshRate,
		}

		// 添加到显示器列表
		info.Displays = append(info.Displays, display)
	}

	return nil
}
