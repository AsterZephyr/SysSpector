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

// 定义WMI查询结构体
type win32Battery struct {
	BatteryStatus       uint16
	EstimatedChargeRemaining uint16
	DesignVoltage       uint32
	FullChargeCapacity  uint32
	Name                string
}

type win32ACAdapter struct {
	DeviceID     string
	Name         string
	Manufacturer string
	Description  string
	Status       string
}

type win32TemperatureSensor struct {
	Name        string
	CurrentReading uint32
	Location    string
}

type win32VideoController struct {
	Name                 string
	Description          string
	AdapterRAM           uint64
	DriverVersion        string
	VideoProcessor       string
	VideoModeDescription string
	CurrentRefreshRate   uint32
	CurrentHorizontalResolution uint32
	CurrentVerticalResolution uint32
	Manufacturer         string
}

type win32DesktopMonitor struct {
	Name                 string
	MonitorManufacturer  string
	MonitorType          string
	ScreenHeight         uint32
	ScreenWidth          uint32
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
	tempInfo, err := getTemperatureInfo()
	if err != nil {
		log.Printf("Error getting temperature info: %v", err)
	} else {
		info.Temperature = tempInfo
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

// getBatteryInfo 获取电池信息
func getBatteryInfo() (model.BatteryInfo, error) {
	var batteryInfo model.BatteryInfo
	
	// 通过WMI查询电池信息
	var batteries []win32Battery
	err := safeWMIQuery("SELECT BatteryStatus, EstimatedChargeRemaining, DesignVoltage, FullChargeCapacity, Name FROM Win32_Battery", &batteries)
	
	if err != nil || len(batteries) == 0 {
		// 尝试使用PowerShell命令获取电池信息
		cmd := exec.Command("powershell", "-Command", "Get-WmiObject -Class Win32_Battery | Select-Object BatteryStatus, EstimatedChargeRemaining, Name")
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

// getTemperatureInfo 获取温度信息
func getTemperatureInfo() ([]model.TempSensorInfo, error) {
	var tempInfo []model.TempSensorInfo
	
	// 首先尝试使用Win32_TemperatureProbe类获取温度信息
	var tempProbes []struct {
		Name             string
		CurrentReading   int32
		Description      string
		DeviceID         string
		Status           string
		LocationInstance string
	}
	err := safeWMIQuery("SELECT Name, CurrentReading, Description, DeviceID, Status, LocationInstance FROM Win32_TemperatureProbe", &tempProbes)
	
	if err == nil && len(tempProbes) > 0 {
		for _, probe := range tempProbes {
			// Win32_TemperatureProbe的温度通常以摄氏度的10倍表示
			temp := float64(probe.CurrentReading) / 10.0
			
			// 跳过无效的温度值
			if temp <= 0 || temp > 120 {
				continue
			}
			
			location := "System"
			if strings.Contains(strings.ToLower(probe.Name), "cpu") || 
			   strings.Contains(strings.ToLower(probe.Description), "cpu") {
				location = "Processor"
			} else if strings.Contains(strings.ToLower(probe.Name), "gpu") || 
			          strings.Contains(strings.ToLower(probe.Description), "gpu") {
				location = "Graphics"
			}
			
			tempInfo = append(tempInfo, model.TempSensorInfo{
				Name:        probe.Name,
				Temperature: temp,
				Location:    location,
				Sensor:      probe.DeviceID,
				Value:       temp,
			})
		}
		
		if len(tempInfo) > 0 {
			return tempInfo, nil
		}
	}
	
	// 如果Win32_TemperatureProbe不可用，尝试使用Win32_TemperatureSensor
	var sensors []win32TemperatureSensor
	err = safeWMIQuery("SELECT Name, CurrentReading, Location FROM Win32_TemperatureSensor", &sensors)
	
	if err == nil && len(sensors) > 0 {
		for _, sensor := range sensors {
			// WMI温度通常以摄氏度的10倍表示
			temp := float64(sensor.CurrentReading) / 10.0
			
			// 跳过无效的温度值
			if temp <= 0 || temp > 120 {
				continue
			}
			
			tempInfo = append(tempInfo, model.TempSensorInfo{
				Name:        sensor.Name,
				Temperature: temp,
				Location:    sensor.Location,
				Sensor:      sensor.Name,
				Value:       temp,
			})
		}
		
		if len(tempInfo) > 0 {
			return tempInfo, nil
		}
	}
	
	// 尝试使用OpenHardwareMonitor获取温度信息
	// 注意：这需要用户安装OpenHardwareMonitor
	ohwmPath := "C:\\Program Files\\OpenHardwareMonitor\\OpenHardwareMonitor.exe"
	cmd := exec.Command(ohwmPath, "/report")
	output, err := cmd.Output()
	
	if err == nil {
		// 解析OpenHardwareMonitor输出
		outputStr := string(output)
		lines := strings.Split(outputStr, "\n")
		
		inTempSection := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			
			if strings.Contains(line, "Temperatures:") {
				inTempSection = true
				continue
			}
			
			if inTempSection && line == "" {
				inTempSection = false
				continue
			}
			
			if inTempSection {
				fields := regexp.MustCompile(`\s+`).Split(line, -1)
				if len(fields) >= 2 {
					tempStr := fields[len(fields)-1]
					tempStr = strings.TrimSuffix(tempStr, "°C")
					temp, err := strconv.ParseFloat(tempStr, 64)
					if err == nil {
						// 确定温度传感器位置
						location := "System"
						if strings.Contains(strings.ToLower(fields[0]), "cpu") {
							location = "Processor"
						} else if strings.Contains(strings.ToLower(fields[0]), "gpu") {
							location = "Graphics"
						}
						
						tempInfo = append(tempInfo, model.TempSensorInfo{
							Name:        fields[0],
							Temperature: temp,
							Location:    location,
							Sensor:      fields[0],
							Value:       temp,
						})
					}
				}
			}
		}
		
		if len(tempInfo) > 0 {
			return tempInfo, nil
		}
	}
	
	// 尝试使用sysctl命令获取CPU温度（适用于某些系统）
	cmd = exec.Command("powershell", "-Command", "Get-WmiObject MSAcpi_ThermalZoneTemperature -Namespace \"root/wmi\"")
	output, err = cmd.Output()
	
	if err == nil {
		outputStr := string(output)
		// 查找温度值
		tempRegex := regexp.MustCompile(`CurrentTemperature\s*:\s*(\d+)`)
		matches := tempRegex.FindStringSubmatch(outputStr)
		
		if len(matches) > 1 {
			// ACPI温度通常以摄氏度的10分之1表示，需要转换
			tempRaw, _ := strconv.ParseInt(matches[1], 10, 64)
			temp := float64(tempRaw - 2732) / 10.0 // 转换开尔文到摄氏度
			
			if temp > 0 && temp < 120 {
				tempInfo = append(tempInfo, model.TempSensorInfo{
					Name:        "CPU",
					Temperature: temp,
					Location:    "Processor",
					Sensor:      "ACPI",
					Value:       temp,
				})
				
				return tempInfo, nil
			}
		}
	}
	
	// 如果以上方法都失败，使用CPU利用率作为近似值
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		// 这只是一个非常粗略的近似值
		// 假设CPU温度与利用率成正比，基准温度为30°C，最高温度为90°C
		estimatedTemp := 30.0 + (cpuPercent[0] * 0.6)
		
		tempInfo = append(tempInfo, model.TempSensorInfo{
			Name:        "CPU",
			Temperature: estimatedTemp,
			Location:    "Processor",
			Sensor:      "Estimated",
			Value:       estimatedTemp,
		})
		
		return tempInfo, nil
	}
	
	// 如果所有方法都失败，返回默认值
	tempInfo = append(tempInfo, model.TempSensorInfo{
		Name:        "CPU",
		Temperature: 0.0,
		Location:    "Processor",
		Sensor:      "Unknown",
		Value:       0.0,
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
		Name             string
		Description      string
		MACAddress       string
		AdapterType      string
		DeviceID         string
		NetConnectionID  string
		Manufacturer     string
		ProductName      string
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
	var videoControllers []win32VideoController
	err := safeWMIQuery("SELECT Name, Description, AdapterRAM, DriverVersion, VideoProcessor, VideoModeDescription, Manufacturer FROM Win32_VideoController", &videoControllers)
	if err != nil {
		return nil, fmt.Errorf("查询显卡信息失败: %v", err)
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
		return nil, fmt.Errorf("查询显示器信息失败: %v", err)
	}

	// 查询显卡信息以获取刷新率
	var videoControllers []win32VideoController
	err = safeWMIQuery("SELECT Name, CurrentRefreshRate, CurrentHorizontalResolution, CurrentVerticalResolution FROM Win32_VideoController", &videoControllers)
	if err != nil {
		return nil, fmt.Errorf("查询显卡信息失败: %v", err)
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
