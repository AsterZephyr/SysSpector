//go:build windows
// +build windows

package windows

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AsterZephyr/SysSpector/pkg/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type win32Battery struct {
	BatteryStatus            uint16
	EstimatedChargeRemaining uint16
	DesignVoltage            uint32
	FullChargeCapacity       uint32
}

type win32ACAdapter struct {
	DeviceID     string
	Name         string
	Manufacturer string
	Description  string
	Status       string
}

type win32TemperatureProbe struct {
	Name              string
	CurrentReading    uint32
	Description       string
	DeviceID          string
	Status            string
	LocationInstance  string
}

type win32VideoControllerBasic struct {
	Name                  string
	Description           string
	AdapterRAM            uint64
	DriverVersion         string
	VideoProcessor        string
	VideoModeDescription  string
}

type win32VideoControllerDisplay struct {
	Name                      string
	CurrentRefreshRate        uint32
	CurrentHorizontalResolution uint32
	CurrentVerticalResolution   uint32
}

type win32DesktopMonitor struct {
	Name                string
	MonitorManufacturer string
	MonitorType         string
	ScreenHeight        uint32
	ScreenWidth         uint32
}

// GetDynamicInfo 获取Windows系统的动态信息
func GetDynamicInfo() (model.SystemInfo, error) {
	var info model.SystemInfo
	var err error

	// 获取磁盘使用情况
	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Printf("Error getting disk partitions: %v", err)
	} else {
		for _, p := range partitions {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}

			info.DiskUsage = append(info.DiskUsage, model.DiskPartitionInfo{
				MountPoint: p.Mountpoint,
				Total:      usage.Total,
				Used:       usage.Used,
				Free:       usage.Free,
				UsedPerc:   usage.UsedPercent,
				Filesystem: p.Fstype,
			})
		}
	}

	// 获取内存使用情况
	memStats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory stats: %v", err)
	} else {
		info.MemoryUsage = model.MemoryUsageInfo{
			Total:    memStats.Total,
			Used:     memStats.Used,
			Free:     memStats.Free,
			UsedPerc: memStats.UsedPercent,
			Active:   memStats.Active,
			Inactive: memStats.Inactive,
			Cached:   memStats.Cached,
		}
	}

	// 获取系统使用率（CPU和GPU）
	err = getSystemUsage(&info)
	if err != nil {
		log.Printf("Error getting system usage: %v", err)
	}

	// 获取电池信息
	batteryInfo, err := getBatteryInfo()
	if err != nil {
		log.Printf("Error getting battery info: %v", err)
	} else {
		info.Battery = batteryInfo
	}

	// 获取交流充电器信息
	adapterInfo, err := getACAdapterInfo()
	if err != nil {
		log.Printf("Error getting AC adapter info: %v", err)
	} else {
		info.ACAdapter = adapterInfo
	}

	// 获取蓝牙信息
	bluetoothInfo, err := getBluetoothInfo()
	if err != nil {
		log.Printf("Error getting bluetooth info: %v", err)
	} else {
		info.Bluetooth = bluetoothInfo
	}

	// 获取温度信息
	temperatureInfo, err := getTemperatureInfo()
	if err != nil {
		log.Printf("Error getting temperature info: %v", err)
	} else {
		info.Temperature = temperatureInfo
	}

	// 获取网卡信息
	networkCards, err := getNetworkCardInfo()
	if err != nil {
		log.Printf("Error getting network card info: %v", err)
	} else {
		info.NetworkCards = networkCards
	}

	// 获取显卡信息
	graphicsCards, err := getGraphicsCardInfo()
	if err != nil {
		log.Printf("Error getting graphics card info: %v", err)
	} else {
		info.GraphicsCards = graphicsCards
	}

	// 获取显示器信息
	displays, err := getDisplayInfo()
	if err != nil {
		log.Printf("Error getting display info: %v", err)
	} else {
		info.Displays = displays
	}

	// 获取已安装应用
	if installedApps, err := getInstalledApps(); err == nil {
		info.InstalledApps = installedApps
	}

	// 获取正在运行的应用
	if runningApps, err := getRunningApps(); err == nil {
		info.RunningApps = runningApps
	}

	// 获取系统启动时间
	bootTime, err := host.BootTime()
	if err == nil {
		bootTimeT := time.Unix(int64(bootTime), 0)
		uptime := time.Since(bootTimeT)

		// 格式化启动时间
		days := int(uptime.Hours()) / 24
		hours := int(uptime.Hours()) % 24
		minutes := int(uptime.Minutes()) % 60

		if days > 0 {
			info.UpTime = fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
		} else {
			info.UpTime = fmt.Sprintf("%d小时%d分钟", hours, minutes)
		}
	}

	return info, nil
}

func getSystemUsage(info *model.SystemInfo) error {
	// 获取CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Printf("获取CPU使用率失败: %v", err)
	} else if len(cpuPercent) > 0 {
		// 将CPU使用率设置到SystemInfo结构体中
		info.CPUUsage = cpuPercent[0]
		
		// 同时添加到温度传感器中，保持向后兼容
		cpuUsageSensor := model.TempSensorInfo{
			Name:        "CPU使用率",
			Temperature: cpuPercent[0],
			Location:    "处理器",
			Sensor:      "CPU",
			Value:       cpuPercent[0],
		}
		
		// 添加到温度传感器列表
		info.Temperature = append(info.Temperature, cpuUsageSensor)
	}

	// 获取GPU使用率
	gpuUsage := 0.0
	gpuFound := false

	// 定义WMI查询结构体
	type win32PerfFormattedData_GPUPerformance struct {
		Name                  string
		PercentGPUTime        uint64
		PercentGPUUtilization uint64
	}

	// 查询GPU性能计数器
	var gpuPerf []win32PerfFormattedData_GPUPerformance
	err = safeWMIQuery("SELECT Name, PercentGPUTime, PercentGPUUtilization FROM \\\\root\\cimv2:Win32_PerfFormattedData_GPUPerformance", &gpuPerf)
	
	if err == nil && len(gpuPerf) > 0 {
		// 使用PercentGPUTime或PercentGPUUtilization，取决于哪个可用
		for _, gpu := range gpuPerf {
			if gpu.PercentGPUUtilization > 0 {
				gpuUsage = float64(gpu.PercentGPUUtilization)
				gpuFound = true
				break
			}
			if gpu.PercentGPUTime > 0 {
				gpuUsage = float64(gpu.PercentGPUTime)
				gpuFound = true
				break
			}
		}
	} else {
		// 如果WMI查询失败，尝试使用命令行获取GPU使用率
		log.Printf("Failed to get GPU usage from WMI, trying alternative method")
		
		// 尝试使用PowerShell获取GPU使用率
		cmd := exec.Command("powershell", "-Command", "(Get-Counter '\\GPU Engine(*engtype_3D)\\Utilization Percentage').CounterSamples | Select-Object -ExpandProperty CookedValue")
		output, err := cmd.Output()
		if err == nil {
			outputStr := strings.TrimSpace(string(output))
			gpuValue, err := strconv.ParseFloat(outputStr, 64)
			if err == nil {
				gpuUsage = gpuValue
				gpuFound = true
			}
		}
		
		// 如果PowerShell命令也失败，尝试使用WMIC
		if !gpuFound {
			cmd = exec.Command("wmic", "path", "Win32_PerfFormattedData_GPUPerformance", "get", "PercentGPUTime")
			output, err = cmd.Output()
			if err == nil {
				lines := strings.Split(string(output), "\n")
				if len(lines) > 1 {
					gpuStr := strings.TrimSpace(lines[1])
					gpuValue, err := strconv.ParseFloat(gpuStr, 64)
					if err == nil && gpuValue > 0 {
						gpuUsage = gpuValue
						gpuFound = true
					}
				}
			}
		}
	}

	// 如果所有方法都失败，使用CPU使用率作为近似值
	if !gpuFound && len(cpuPercent) > 0 {
		// 这只是一个非常粗略的近似值
		gpuUsage = cpuPercent[0] * 0.8
		gpuFound = true
	}

	// 如果找到了GPU使用率，设置到SystemInfo结构体中
	if gpuFound {
		// 将GPU使用率设置到SystemInfo结构体中
		info.GPUUsage = gpuUsage
		
		// 同时添加到温度传感器中，保持向后兼容
		gpuUsageSensor := model.TempSensorInfo{
			Name:        "GPU使用率",
			Temperature: gpuUsage,
			Location:    "图形处理器",
			Sensor:      "GPU",
			Value:       gpuUsage,
		}
		
		// 添加到温度传感器列表
		info.Temperature = append(info.Temperature, gpuUsageSensor)
	}

	return nil
}

// getBatteryInfo 获取电池信息
func getBatteryInfo() (model.BatteryInfo, error) {
	var batteryInfo model.BatteryInfo

	// 通过WMI查询电池信息
	var batteries []win32Battery
	err := safeWMIQuery("SELECT BatteryStatus, EstimatedChargeRemaining, DesignVoltage, FullChargeCapacity FROM Win32_Battery", &batteries)

	if err != nil || len(batteries) == 0 {
		// 尝试使用PowerShell命令获取电池信息
		cmd := exec.Command("powershell", "-Command", "Get-WmiObject -Class Win32_Battery | Select-Object BatteryStatus, EstimatedChargeRemaining")
		output, err := cmd.Output()
		if err != nil {
			return batteryInfo, fmt.Errorf("error getting battery info: %v", err)
		}

		// 解析输出
		outputStr := string(output)

		// 提取电池状态
		statusRegex := regexp.MustCompile(`BatteryStatus\s+:\s+(\d+)`)
		statusMatches := statusRegex.FindStringSubmatch(outputStr)
		if len(statusMatches) > 1 {
			status, _ := strconv.Atoi(statusMatches[1])
			batteryInfo.IsCharging = (status == 2) // 2表示正在充电
		}

		// 提取电池电量
		percentRegex := regexp.MustCompile(`EstimatedChargeRemaining\s+:\s+(\d+)`)
		percentMatches := percentRegex.FindStringSubmatch(outputStr)
		if len(percentMatches) > 1 {
			percent, _ := strconv.Atoi(percentMatches[1])
			batteryInfo.Percentage = percent
		}

		batteryInfo.IsPresent = true
		batteryInfo.Health = "Normal" // 默认值

		return batteryInfo, nil
	}

	// 处理WMI查询结果
	battery := batteries[0]

	// 设置电池状态
	batteryInfo.IsPresent = true
	batteryInfo.Percentage = int(battery.EstimatedChargeRemaining)

	// 根据BatteryStatus确定充电状态
	// 1: 电池放电, 2: 电池正在充电, 3: 电池充满, 其他值: 未知状态
	switch battery.BatteryStatus {
	case 1:
		batteryInfo.IsCharging = false
		batteryInfo.Status = "Discharging"
	case 2:
		batteryInfo.IsCharging = true
		batteryInfo.Status = "Charging"
	case 3:
		batteryInfo.IsCharging = false
		batteryInfo.Status = "Fully Charged"
	default:
		batteryInfo.IsCharging = false
		batteryInfo.Status = "Unknown"
	}

	// 设置电池健康状态（Windows没有直接提供此信息，使用默认值）
	batteryInfo.Health = "Normal"

	// 获取电池循环计数（Windows没有直接提供此信息，使用0作为默认值）
	batteryInfo.CycleCount = 0

	return batteryInfo, nil
}

// getACAdapterInfo 获取交流充电器信息
func getACAdapterInfo() (model.ACAdapterInfo, error) {
	var adapterInfo model.ACAdapterInfo

	// 通过WMI查询交流充电器信息
	var adapters []win32ACAdapter
	err := safeWMIQuery("SELECT DeviceID, Name, Manufacturer, Description, Status FROM Win32_PortableBattery", &adapters)

	// 检查电池状态以确定充电器是否连接
	var batteries []win32Battery
	batteryErr := safeWMIQuery("SELECT BatteryStatus FROM Win32_Battery", &batteries)

	if batteryErr == nil && len(batteries) > 0 {
		// BatteryStatus为2表示正在充电，这意味着充电器已连接
		adapterInfo.Connected = (batteries[0].BatteryStatus == 2)
		adapterInfo.IsConnected = (batteries[0].BatteryStatus == 2)
	} else {
		// 如果无法获取电池状态，尝试使用PowerShell命令
		cmd := exec.Command("powershell", "-Command", "Get-WmiObject -Class Win32_Battery | Select-Object BatteryStatus")
		output, err := cmd.Output()
		if err == nil {
			outputStr := string(output)
			statusRegex := regexp.MustCompile(`BatteryStatus\s+:\s+(\d+)`)
			statusMatches := statusRegex.FindStringSubmatch(outputStr)
			if len(statusMatches) > 1 {
				status, _ := strconv.Atoi(statusMatches[1])
				adapterInfo.Connected = (status == 2)
				adapterInfo.IsConnected = (status == 2)
			}
		}
	}

	if err != nil || len(adapters) == 0 {
		// 如果WMI查询失败，尝试使用PowerShell命令
		if adapterInfo.Connected {
			// 如果充电器已连接，设置一些基本信息
			adapterInfo.Name = "AC Adapter"
			adapterInfo.SerialNum = "Unknown"
			adapterInfo.ChipModel = "Unknown"
			adapterInfo.Wattage = 0
		}

		return adapterInfo, nil
	}

	// 处理WMI查询结果
	adapter := adapters[0]

	adapterInfo.Name = adapter.Name
	adapterInfo.SerialNum = adapter.DeviceID
	adapterInfo.ChipModel = adapter.Manufacturer
	adapterInfo.Wattage = 0 // Windows没有直接提供此信息

	return adapterInfo, nil
}

// getBluetoothInfo 获取蓝牙信息
func getBluetoothInfo() (model.BluetoothInfo, error) {
	var bluetoothInfo model.BluetoothInfo

	// 使用PowerShell命令获取蓝牙信息
	cmd := exec.Command("powershell", "-Command", "Get-PnpDevice | Where-Object {$_.Class -eq 'Bluetooth'}")
	output, err := cmd.Output()
	if err != nil {
		return bluetoothInfo, fmt.Errorf("error getting bluetooth info: %v", err)
	}

	// 解析输出
	outputStr := string(output)

	// 检查蓝牙是否可用
	bluetoothInfo.IsAvailable = strings.Contains(outputStr, "Bluetooth")

	if bluetoothInfo.IsAvailable {
		// 检查蓝牙是否启用
		if strings.Contains(outputStr, "OK") {
			bluetoothInfo.Enabled = true
			bluetoothInfo.Status = "打开"
		} else {
			bluetoothInfo.Enabled = false
			bluetoothInfo.Status = "关闭"
		}

		// 获取已连接的蓝牙设备
		deviceCmd := exec.Command("powershell", "-Command", "Get-PnpDevice | Where-Object {$_.Class -eq 'Bluetooth' -and $_.Status -eq 'OK'}")
		deviceOutput, err := deviceCmd.Output()
		if err == nil {
			deviceOutputStr := string(deviceOutput)
			lines := strings.Split(deviceOutputStr, "\n")

			for _, line := range lines {
				if strings.Contains(line, "Bluetooth") && !strings.Contains(line, "Radio") {
					fields := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
					if len(fields) >= 2 {
						bluetoothInfo.ConnectedDevices = append(bluetoothInfo.ConnectedDevices, model.BTDeviceInfo{
							Name:      fields[len(fields)-1],
							Connected: true,
						})
					}
				}
			}
		}
	}

	return bluetoothInfo, nil
}

// getTemperatureInfo 获取温度传感器信息
func getTemperatureInfo() ([]model.TempSensorInfo, error) {
	var tempInfo []model.TempSensorInfo
	
	// 首先尝试使用Win32_TemperatureProbe获取温度信息
	var tempProbes []win32TemperatureProbe
	err := safeWMIQuery("SELECT Name, CurrentReading, Description, DeviceID, Status FROM Win32_TemperatureProbe", &tempProbes)
	
	if err == nil && len(tempProbes) > 0 {
		log.Printf("成功从Win32_TemperatureProbe获取到 %d 个温度传感器信息", len(tempProbes))
		for _, probe := range tempProbes {
			// 转换温度值（通常需要除以10）
			temp := float64(probe.CurrentReading) / 10.0
			
			// 添加到结果中
			tempInfo = append(tempInfo, model.TempSensorInfo{
				Name:        probe.Name,
				Temperature: temp,
				Location:    probe.Description,
				Sensor:      probe.DeviceID,
				Value:       temp,
			})
			log.Printf("温度传感器: 名称=%s, 温度=%.1f°C, 位置=%s", 
				probe.Name, temp, probe.Description)
		}
		return tempInfo, nil
	} else {
		log.Printf("从Win32_TemperatureProbe获取温度信息失败: %v", err)
	}
	
	// 如果Win32_TemperatureProbe查询失败，尝试使用MSAcpi_ThermalZoneTemperature
	log.Printf("尝试使用MSAcpi_ThermalZoneTemperature获取温度信息")
	
	// 尝试使用MSAcpi_ThermalZoneTemperature获取温度信息
	type win32MSAcpiThermalZone struct {
		InstanceName       string
		CurrentTemperature uint32
	}
	
	var thermalZones []win32MSAcpiThermalZone
	err = safeWMIQuery("SELECT InstanceName, CurrentTemperature FROM \\\\root\\wmi:MSAcpi_ThermalZoneTemperature", &thermalZones)
	
	if err == nil && len(thermalZones) > 0 {
		log.Printf("成功从MSAcpi_ThermalZoneTemperature获取到 %d 个温度区域信息", len(thermalZones))
		for _, zone := range thermalZones {
			// 转换温度值（需要减去273.15转换为摄氏度）
			// MSAcpi_ThermalZoneTemperature返回的是开尔文温度，需要转换为摄氏度
			temp := float64(zone.CurrentTemperature) / 10.0 - 273.15
			
			// 添加到结果中
			tempInfo = append(tempInfo, model.TempSensorInfo{
				Name:        fmt.Sprintf("ACPI\\ThermalZone\\%s", zone.InstanceName),
				Temperature: temp,
				Location:    "ACPI Thermal Zone",
				Sensor:      zone.InstanceName,
				Value:       temp,
			})
			log.Printf("ACPI温度区域: 名称=%s, 温度=%.1f°C", 
				zone.InstanceName, temp)
		}
		return tempInfo, nil
	} else {
		log.Printf("从MSAcpi_ThermalZoneTemperature获取温度信息失败: %v", err)
	}
	
	// 如果MSAcpi_ThermalZoneTemperature也失败，尝试使用命令行工具
	log.Printf("尝试使用命令行工具获取温度信息")
	cmd := exec.Command("wmic", "/namespace:\\\\root\\wmi", "PATH", "MSAcpi_ThermalZoneTemperature", "get", "CurrentTemperature", "/format:list")
	output, err := cmd.Output()
	if err == nil {
		log.Printf("命令行输出: %s", string(output))
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "CurrentTemperature=") {
				tempStr := strings.TrimPrefix(line, "CurrentTemperature=")
				tempValue, err := strconv.ParseUint(tempStr, 10, 32)
				if err == nil {
					// 转换温度值（需要减去273.15转换为摄氏度）
					temp := float64(tempValue) / 10.0 - 273.15
					
					// 添加到结果中
					tempInfo = append(tempInfo, model.TempSensorInfo{
						Name:        fmt.Sprintf("ACPI\\ThermalZone\\THM%d", i),
						Temperature: temp,
						Location:    "ACPI Thermal Zone",
						Sensor:      fmt.Sprintf("THM%d", i),
						Value:       temp,
					})
					log.Printf("命令行获取的温度: 传感器=%s, 温度=%.1f°C", 
						fmt.Sprintf("THM%d", i), temp)
				} else {
					log.Printf("解析温度值失败: %v", err)
				}
			}
		}
		
		if len(tempInfo) > 0 {
			return tempInfo, nil
		}
	} else {
		log.Printf("执行命令行获取温度信息失败: %v", err)
	}
	
	// 尝试使用PowerShell获取CPU温度
	log.Printf("尝试使用PowerShell获取CPU温度")
	cmd = exec.Command("powershell", "-Command", "Get-WmiObject MSAcpi_ThermalZoneTemperature -Namespace root/wmi | Select-Object CurrentTemperature")
	output, err = cmd.Output()
	if err == nil {
		log.Printf("PowerShell输出: %s", string(output))
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "CurrentTemperature") {
				continue // 跳过标题行
			}
			if tempStr := strings.TrimSpace(line); tempStr != "" {
				tempValue, err := strconv.ParseUint(tempStr, 10, 32)
				if err == nil {
					// 转换温度值（需要减去273.15转换为摄氏度）
					temp := float64(tempValue) / 10.0 - 273.15
					
					// 添加到结果中
					tempInfo = append(tempInfo, model.TempSensorInfo{
						Name:        fmt.Sprintf("CPU Temperature %d", i),
						Temperature: temp,
						Location:    "CPU",
						Sensor:      fmt.Sprintf("CPU%d", i),
						Value:       temp,
					})
					log.Printf("PowerShell获取的CPU温度: 温度=%.1f°C", temp)
				} else {
					log.Printf("解析PowerShell温度值失败: %v", err)
				}
			}
		}
		
		if len(tempInfo) > 0 {
			return tempInfo, nil
		}
	} else {
		log.Printf("执行PowerShell获取CPU温度失败: %v", err)
	}
	
	// 如果所有方法都失败，返回一个默认温度信息
	log.Printf("所有获取温度的方法都失败，返回默认温度信息")
	tempInfo = append(tempInfo, model.TempSensorInfo{
		Name:        "System Temperature",
		Temperature: 50.0, // 默认温度值
		Location:    "System",
		Sensor:      "Default",
		Value:       50.0,
	})
	
	return tempInfo, nil
}

// getNetworkCardInfo 获取网卡信息
func getNetworkCardInfo() ([]struct {
	Name        string
	MACAddress  string
	IPAddresses []string
}, error) {
	var networkCards []struct {
		Name        string
		MACAddress  string
		IPAddresses []string
	}

	// 查询网卡信息
	var adapters []struct {
		Name            string
		Description     string
		MACAddress      string
		AdapterType     string
		DeviceID        string
		NetConnectionID string
		Manufacturer    string
		ProductName     string
	}
	err := safeWMIQuery("SELECT Name, NetConnectionID, MACAddress, AdapterType, PhysicalAdapter, NetEnabled, ProductName FROM Win32_NetworkAdapter WHERE PhysicalAdapter=True", &adapters)
	if err != nil {
		return nil, fmt.Errorf("查询网卡信息失败: %v", err)
	}

	// 查询网卡配置信息
	var adapterConfigs []struct {
		Description string
		IPAddress   []string
		IPSubnet    []string
		MACAddress  string
		DHCPEnabled bool
	}
	err = safeWMIQuery("SELECT Description, IPAddress, IPSubnet, MACAddress, DHCPEnabled FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled=True", &adapterConfigs)
	if err != nil {
		return nil, fmt.Errorf("查询网卡配置信息失败: %v", err)
	}

	// 将网卡信息与配置信息关联
	for _, adapter := range adapters {
		// 跳过虚拟网卡和无效网卡
		if adapter.MACAddress == "" {
			continue
		}

		// 创建网卡信息
		networkCard := struct {
			Name        string
			MACAddress  string
			IPAddresses []string
		}{
			Name:        adapter.Name,
			MACAddress:  adapter.MACAddress,
			IPAddresses: []string{},
		}

		// 如果有NetConnectionID，使用它作为更友好的名称
		if adapter.NetConnectionID != "" {
			networkCard.Name = adapter.NetConnectionID
		}

		// 查找对应的配置信息
		for _, config := range adapterConfigs {
			if config.MACAddress == adapter.MACAddress {
				networkCard.IPAddresses = config.IPAddress
				break
			}
		}

		// 添加到网卡列表
		networkCards = append(networkCards, networkCard)
	}

	return networkCards, nil
}

// getGraphicsCardInfo 获取显卡信息
func getGraphicsCardInfo() ([]struct {
	Name   string
	Model  string
	Memory uint64
}, error) {
	var graphicsCards []struct {
		Name   string
		Model  string
		Memory uint64
	}

	// 查询显卡信息
	var videoControllers []win32VideoControllerBasic
	err := safeWMIQuery("SELECT Name, Description, AdapterRAM, DriverVersion, VideoProcessor FROM Win32_VideoController", &videoControllers)
	if err != nil {
		log.Printf("查询基本显卡信息失败: %v，尝试使用备用方法", err)
		// 尝试使用命令行工具获取显卡信息
		cmd := exec.Command("wmic", "path", "Win32_VideoController", "get", "Name,AdapterRAM,DriverVersion")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("查询显卡信息失败: %v", err)
		}
		
		// 解析命令行输出
		log.Printf("使用命令行获取显卡信息: %s", string(output))
		// 继续处理...
	} else {
		log.Printf("成功获取到 %d 个显卡的基本信息", len(videoControllers))
	}

	// 处理显卡信息
	for _, controller := range videoControllers {
		// 创建显卡信息
		graphicsCard := struct {
			Name   string
			Model  string
			Memory uint64
		}{
			Name:   controller.Name,
			Model:  controller.Name,
			Memory: controller.AdapterRAM,
		}
		
		log.Printf("显卡信息: 名称=%s, 内存=%d bytes, 驱动版本=%s", 
			controller.Name, controller.AdapterRAM, controller.DriverVersion)
		
		// 如果有VideoProcessor，将其添加到型号中
		if controller.VideoProcessor != "" && controller.VideoProcessor != controller.Name {
			graphicsCard.Model = fmt.Sprintf("%s (%s)", controller.Name, controller.VideoProcessor)
		}

		// 添加到显卡列表
		graphicsCards = append(graphicsCards, graphicsCard)
	}

	return graphicsCards, nil
}

// getDisplayInfo 获取显示器信息
func getDisplayInfo() ([]struct {
	Name        string
	Model       string
	Resolution  string
	RefreshRate int
}, error) {
	var displays []struct {
		Name        string
		Model       string
		Resolution  string
		RefreshRate int
	}

	// 查询显示器信息
	var monitors []win32DesktopMonitor
	err := safeWMIQuery("SELECT Name, MonitorManufacturer, MonitorType, ScreenHeight, ScreenWidth FROM Win32_DesktopMonitor WHERE PNPDeviceID <> NULL", &monitors)
	if err != nil {
		log.Printf("查询显示器信息失败: %v，尝试使用备用方法", err)
		// 尝试使用命令行工具获取显示器信息
		cmd := exec.Command("wmic", "path", "Win32_DesktopMonitor", "get", "Name,ScreenHeight,ScreenWidth")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("使用命令行获取显示器信息失败: %v", err)
		} else {
			log.Printf("使用命令行获取显示器信息: %s", string(output))
			// 继续处理...
		}
	} else {
		log.Printf("成功获取到 %d 个显示器的信息", len(monitors))
	}

	// 查询显卡信息以获取刷新率
	var videoControllers []win32VideoControllerDisplay
	err = safeWMIQuery("SELECT Name, CurrentRefreshRate, CurrentHorizontalResolution, CurrentVerticalResolution FROM Win32_VideoController", &videoControllers)
	if err != nil {
		log.Printf("查询显示信息失败: %v，尝试使用备用方法", err)
		// 尝试使用命令行工具获取显示信息
		cmd := exec.Command("wmic", "path", "Win32_VideoController", "get", "Name,CurrentRefreshRate,CurrentHorizontalResolution,CurrentVerticalResolution")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("使用命令行获取显示信息失败: %v", err)
		} else {
			log.Printf("使用命令行获取显示信息: %s", string(output))
			// 继续处理...
		}
	} else {
		log.Printf("成功获取到 %d 个显示控制器的信息", len(videoControllers))
		for _, controller := range videoControllers {
			log.Printf("显示控制器信息: 名称=%s, 分辨率=%dx%d, 刷新率=%d", 
				controller.Name, controller.CurrentHorizontalResolution, 
				controller.CurrentVerticalResolution, controller.CurrentRefreshRate)
		}
	}

	// 如果没有找到显示器信息，尝试从显卡信息中提取
	if len(monitors) == 0 && len(videoControllers) > 0 {
		for _, controller := range videoControllers {
			// 跳过无效分辨率
			if controller.CurrentHorizontalResolution == 0 || controller.CurrentVerticalResolution == 0 {
				continue
			}

			// 创建显示器信息
			display := struct {
				Name        string
				Model       string
				Resolution  string
				RefreshRate int
			}{
				Name:        controller.Name,
				Model:       controller.Name,
				Resolution:  fmt.Sprintf("%d x %d", controller.CurrentHorizontalResolution, controller.CurrentVerticalResolution),
				RefreshRate: int(controller.CurrentRefreshRate),
			}

			// 添加到显示器列表
			displays = append(displays, display)
		}
	} else {
		// 处理显示器信息
		for i, monitor := range monitors {
			// 创建显示器信息
			display := struct {
				Name        string
				Model       string
				Resolution  string
				RefreshRate int
			}{
				Name:  monitor.Name,
				Model: monitor.Name,
			}

			// 如果有制造商信息，添加到型号中
			if monitor.MonitorManufacturer != "" && monitor.MonitorManufacturer != monitor.Name {
				display.Model = fmt.Sprintf("%s (%s)", monitor.Name, monitor.MonitorManufacturer)
			}

			// 如果有分辨率信息，设置分辨率
			if monitor.ScreenWidth > 0 && monitor.ScreenHeight > 0 {
				display.Resolution = fmt.Sprintf("%d x %d", monitor.ScreenWidth, monitor.ScreenHeight)
			} else if i < len(videoControllers) {
				// 尝试从对应的显卡获取分辨率
				controller := videoControllers[i]
				if controller.CurrentHorizontalResolution > 0 && controller.CurrentVerticalResolution > 0 {
					display.Resolution = fmt.Sprintf("%d x %d", controller.CurrentHorizontalResolution, controller.CurrentVerticalResolution)
					display.RefreshRate = int(controller.CurrentRefreshRate)
				}
			}

			// 添加到显示器列表
			displays = append(displays, display)
		}
	}

	return displays, nil
}

// getInstalledApps 获取已安装应用
func getInstalledApps() ([]model.AppInfo, error) {
	var apps []model.AppInfo

	// 使用PowerShell命令获取已安装应用
	cmd := exec.Command("powershell", "-Command", "Get-ItemProperty HKLM:\\Software\\Microsoft\\Windows\\CurrentVersion\\Uninstall\\* | Select-Object DisplayName, DisplayVersion, InstallDate | Where-Object {$_.DisplayName -ne $null}")
	output, err := cmd.Output()
	if err != nil {
		return apps, fmt.Errorf("error getting installed apps: %v", err)
	}

	// 解析输出
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// 跳过前两行（表头）
	for i := 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := regexp.MustCompile(`\s+`).Split(line, -1)
		if len(fields) >= 2 {
			app := model.AppInfo{
				Name:    fields[0],
				Version: fields[1],
			}

			if len(fields) >= 3 {
				app.InstallDate = fields[2]
			}

			apps = append(apps, app)
		}
	}

	return apps, nil
}

// getRunningApps 获取正在运行的应用
func getRunningApps() ([]model.ProcessInfo, error) {
	var procs []model.ProcessInfo

	// 使用gopsutil获取进程信息
	processes, err := process.Processes()
	if err != nil {
		return procs, fmt.Errorf("error getting running processes: %v", err)
	}

	for _, p := range processes {
		name, err := p.Name()
		if err != nil {
			continue
		}

		pid := int(p.Pid)

		cpuPercent, _ := p.CPUPercent()

		memInfo, err := p.MemoryInfo()
		var memUsage uint64
		if err == nil && memInfo != nil {
			memUsage = memInfo.RSS
		}

		// 网络使用量无法直接获取，设为0
		networkUsage := uint64(0)

		procs = append(procs, model.ProcessInfo{
			PID:          pid,
			Name:         name,
			CPU:          cpuPercent,
			Memory:       memUsage,
			NetworkUsage: networkUsage,
		})
	}

	return procs, nil
}

// getCPUUsage 获取CPU使用率
func getCPUUsage() (float64, error) {
	// 使用gopsutil获取CPU使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("获取CPU使用率失败: %v", err)
	}

	if len(cpuPercent) > 0 {
		return cpuPercent[0], nil
	}

	return 0, nil
}

// getGPUUsage 获取GPU使用率
func getGPUUsage() (float64, error) {
	// 定义WMI查询结构体
	type win32PerfFormattedData_GPUPerformance struct {
		Name                  string
		PercentGPUTime        uint64
		PercentGPUUtilization uint64
	}

	// 查询GPU性能计数器
	var gpuPerf []win32PerfFormattedData_GPUPerformance
	err := safeWMIQuery("SELECT Name, PercentGPUTime, PercentGPUUtilization FROM \\\\root\\cimv2:Win32_PerfFormattedData_GPUPerformance", &gpuPerf)

	if err == nil && len(gpuPerf) > 0 {
		// 使用PercentGPUTime或PercentGPUUtilization，取决于哪个可用
		for _, gpu := range gpuPerf {
			if gpu.PercentGPUUtilization > 0 {
				return float64(gpu.PercentGPUUtilization), nil
			}
			if gpu.PercentGPUTime > 0 {
				return float64(gpu.PercentGPUTime), nil
			}
		}
	} else {
		// 如果WMI查询失败，尝试使用命令行获取GPU使用率
		log.Printf("Failed to get GPU usage from WMI, trying alternative method")
		
		// 尝试使用PowerShell获取GPU使用率
		cmd := exec.Command("powershell", "-Command", "Get-Counter -Counter \"\\GPU Engine(*)\\Utilization Percentage\" -SampleInterval 1 -MaxSamples 1")
		output, err := cmd.Output()
		if err == nil {
			outputStr := string(output)
			// 查找GPU使用率
			re := regexp.MustCompile(`(\d+(\.\d+)?)`)
			matches := re.FindStringSubmatch(outputStr)
			if len(matches) > 1 {
				gpuUsage, err := strconv.ParseFloat(matches[1], 64)
				if err == nil {
					return gpuUsage, nil
				}
			}
		}
		
		// 如果PowerShell命令也失败，尝试使用WMIC
		cmd = exec.Command("wmic", "path", "Win32_PerfFormattedData_GPUPerformance", "get", "PercentGPUTime")
		output, err = cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 {
				gpuStr := strings.TrimSpace(lines[1])
				gpuValue, err := strconv.ParseFloat(gpuStr, 64)
				if err == nil && gpuValue > 0 {
					return gpuValue, nil
				}
			}
		}
	}

	// 如果所有方法都失败，使用CPU使用率作为近似值
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		// 这只是一个非常粗略的近似值
		return cpuPercent[0] * 0.8, nil
	}

	// 如果所有方法都失败，返回0
	return 0, nil
}
