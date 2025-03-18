//go:build windows
// +build windows

package windows

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/StackExchange/wmi"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// WMI 查询结构体定义
// win32ComputerSystem 表示计算机系统信息
type win32ComputerSystem struct {
	Model               string // 计算机型号
	Name                string // 计算机名称
	TotalPhysicalMemory uint64 // 物理内存总量
}

// win32Processor 表示处理器信息
type win32Processor struct {
	Name          string // 处理器名称
	NumberOfCores uint32 // 处理器核心数
}

// win32BIOS 表示BIOS信息
// 该结构体用于存储BIOS的基本信息，包括序列号
type win32BIOS struct {
	SerialNumber string // 序列号
}

// win32PhysicalMemory 表示物理内存信息
type win32PhysicalMemory struct {
	Capacity   uint64 // 内存容量
	MemoryType uint16 // 内存类型代码
}

// win32DiskDrive 表示磁盘驱动器信息
type win32DiskDrive struct {
	Caption      string // 磁盘标题
	Model        string // 磁盘型号
	Size         string // 磁盘容量
	SerialNumber string // 磁盘序列号
}

// win32ComputerSystemProduct 表示计算机系统产品信息

type win32ComputerSystemProduct struct {
	UUID string // 硬件UUID
}

// safeWMIQuery :对wmi.Query 的安全封装
func safeWMIQuery(query string, dst interface{}) error {
	// 使用当前版本的 wmi.Query 方法
	err := wmi.Query(query, dst)
	if err != nil {
		// 记录错误，但不中断执行
		log.Printf("WMI query failed: %v. Query: %s", err, query)

		// 检查是否是权限或者类不存在的错误
		errStr := err.Error()
		if strings.Contains(errStr, "access denied") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "invalid class") {
			// 这可能是由于 Windows 版本不兼容导致的
			log.Printf("This may be due to Windows version incompatibility")
		}
	}
	return err
}

// GetSystemInfo 收集 Windows 系统的硬件和系统信息
// 该函数用于收集Windows系统的硬件和系统信息，包括主机名、操作系统信息、计算机系统信息、序列号、CPU信息、内存信息、磁盘信息和硬件UUID
func GetSystemInfo() (model.SystemInfo, error) {
	var info model.SystemInfo
	var err error

	// 获取主机名和操作系统信息
	// 通过调用host.Info()函数获取主机名和操作系统信息
	hostInfo, err := host.Info()
	if err != nil {
		log.Printf("Error getting host info: %v", err)
	} else {
		info.Hostname = hostInfo.Hostname
		info.OS = hostInfo.Platform + " " + hostInfo.PlatformVersion

		// 记录 Windows 版本信息，用于后续可能的版本特定查询
		log.Printf("Windows version: %s %s", hostInfo.Platform, hostInfo.PlatformVersion)
	}

	// 获取计算机系统信息
	// 通过调用safeWMIQuery()函数查询Win32_ComputerSystem表获取计算机系统信息
	var computerSystems []win32ComputerSystem
	err = safeWMIQuery("SELECT Model, Name, TotalPhysicalMemory FROM Win32_ComputerSystem", &computerSystems)
	if err == nil && len(computerSystems) > 0 {
		info.ModelID = computerSystems[0].Model // 在Windows中，型号标识符与型号名称相同
		info.Model = computerSystems[0].Model

		// 尝试获取更友好的型号名称
		marketingName, err := getMarketingModelName(info.Model)
		if err == nil && marketingName != "" {
			info.Model = marketingName
		}
	}

	// 获取序列号
	// 通过调用safeWMIQuery()函数查询Win32_BIOS表获取序列号
	var biosInfo []win32BIOS
	err = safeWMIQuery("SELECT SerialNumber FROM Win32_BIOS", &biosInfo)
	if err == nil && len(biosInfo) > 0 {
		info.SerialNumber = biosInfo[0].SerialNumber
	}

	// 获取CPU信息
	// 通过调用safeWMIQuery()函数查询Win32_Processor表获取CPU信息
	var processors []win32Processor
	err = safeWMIQuery("SELECT Name, NumberOfCores FROM Win32_Processor", &processors)
	if err == nil && len(processors) > 0 {
		info.CPU = model.CPUInfo{
			Model: processors[0].Name,
			Cores: int(processors[0].NumberOfCores),
		}
	} else {
		// 如果WMI查询失败，尝试使用备用方法获取CPU信息
		log.Printf("Falling back to alternative method for CPU info")

		// 在某些Windows版本中，Win32_Processor可能有不同的属性名称
		var altProcessors []struct {
			ProcessorName string
			CoreCount     uint32
		}

		// 尝试备选查询
		altErr := safeWMIQuery("SELECT Name AS ProcessorName, NumberOfCores AS CoreCount FROM Win32_Processor", &altProcessors)
		if altErr == nil && len(altProcessors) > 0 {
			info.CPU = model.CPUInfo{
				Model: altProcessors[0].ProcessorName,
				Cores: int(altProcessors[0].CoreCount),
			}
		} else {
			// 如果备选查询也失败，使用默认值
			info.CPU = model.CPUInfo{
				Model: "Unknown CPU",
				Cores: 0,
			}
		}
	}

	// 通过调用safeWMIQuery()函数查询Win32_PhysicalMemory表获取内存信息
	var memoryInfo []win32PhysicalMemory
	err = safeWMIQuery("SELECT Capacity, MemoryType FROM Win32_PhysicalMemory", &memoryInfo)

	// 通过调用mem.VirtualMemory()函数获取内存信息，并计算总内存和内存类型
	memStats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory info: %v", err)
	} else {
		info.Memory = model.MemoryInfo{
			Total: memStats.Total,
			Type:  getMemoryTypeString(memoryInfo),
		}
	}

	// 获取磁盘信息
	// 通过调用safeWMIQuery()函数查询Win32_DiskDrive表获取磁盘信息
	var diskDrives []win32DiskDrive
	err = safeWMIQuery("SELECT Caption, Model, Size, SerialNumber FROM Win32_DiskDrive", &diskDrives)
	if err == nil {
		for _, d := range diskDrives {
			size, _ := strconv.ParseUint(d.Size, 10, 64)
			// 转换为GB
			sizeGB := size / (1024 * 1024 * 1024)
			info.Disks = append(info.Disks, model.Disk{
				Name:   d.Caption,
				Model:  d.Model,
				Size:   sizeGB,
				Serial: d.SerialNumber,
			})
		}
	} else {
		// 如果标准查询失败，尝试备选查询（适用于某些Windows版本）
		var altDiskDrives []struct {
			DiskName   string
			DiskModel  string
			DiskSize   string
			DiskSerial string
		}

		// 尝试备选查询
		altErr := safeWMIQuery("SELECT Caption AS DiskName, Model AS DiskModel, Size AS DiskSize, SerialNumber AS DiskSerial FROM Win32_DiskDrive", &altDiskDrives)
		if altErr == nil {
			for _, d := range altDiskDrives {
				size, _ := strconv.ParseUint(d.DiskSize, 10, 64)
				// 转换为GB
				sizeGB := size / (1024 * 1024 * 1024)
				info.Disks = append(info.Disks, model.Disk{
					Name:   d.DiskName,
					Model:  d.DiskModel,
					Size:   sizeGB,
					Serial: d.DiskSerial,
				})
			}
		}
	}

	// 通过调用safeWMIQuery()函数查询Win32_ComputerSystemProduct表获取硬件UUID
	var systemProducts []win32ComputerSystemProduct
	err = safeWMIQuery("SELECT UUID FROM Win32_ComputerSystemProduct", &systemProducts)
	if err == nil && len(systemProducts) > 0 {
		info.UUID = systemProducts[0].UUID
	}

	return info, nil
}

// getMarketingModelName 尝试获取更友好的型号名称
func getMarketingModelName(modelID string) (string, error) {
	// 从WMI获取更详细的系统信息
	var productInfo []struct {
		Name string
	}

	err := safeWMIQuery("SELECT Name FROM Win32_ComputerSystemProduct", &productInfo)
	if err == nil && len(productInfo) > 0 && productInfo[0].Name != "" {
		return productInfo[0].Name, nil
	}

	// 如果无法从WMI获取，使用映射表，实际应用中可能需要更完整的映射表
	modelMap := map[string]string{
		"Surface Laptop 3":   "Microsoft Surface Laptop 3",
		"Surface Pro 7":      "Microsoft Surface Pro 7",
		"Surface Book 3":     "Microsoft Surface Book 3",
		"XPS 15 9500":        "Dell XPS 15 9500",
		"XPS 13 9310":        "Dell XPS 13 9310",
		"ThinkPad X1 Carbon": "Lenovo ThinkPad X1 Carbon",
		"ThinkPad T14s":      "Lenovo ThinkPad T14s",
		"EliteBook 840 G7":   "HP EliteBook 840 G7",
		"ProBook 450 G7":     "HP ProBook 450 G7",
		"ZBook Studio G7":    "HP ZBook Studio G7",
	}

	// 尝试精确匹配
	if name, ok := modelMap[modelID]; ok {
		return name, nil
	}

	// 尝试部分匹配
	for key, value := range modelMap {
		if strings.Contains(modelID, key) {
			return value, nil
		}
	}

	// 如果在映射表中找不到，则返回原始型号ID
	return modelID, nil
}

// getMemoryTypeString 将WMI内存类型代码转换为字符串描述
func getMemoryTypeString(memoryModules []win32PhysicalMemory) string {
	if len(memoryModules) == 0 {
		return "Unknown"
	}

	// 根据WMI文档的内存类型代码
	memoryTypes := map[uint16]string{
		0:  "Unknown",
		1:  "Other",
		2:  "DRAM",
		3:  "Synchronous DRAM",
		4:  "Cache DRAM",
		5:  "EDO",
		6:  "EDRAM",
		7:  "VRAM",
		8:  "SRAM",
		9:  "RAM",
		10: "ROM",
		11: "Flash",
		12: "EEPROM",
		13: "FEPROM",
		14: "EPROM",
		15: "CDRAM",
		16: "3DRAM",
		17: "SDRAM",
		18: "SGRAM",
		19: "RDRAM",
		20: "DDR",
		21: "DDR2",
		22: "DDR2 FB-DIMM",
		24: "DDR3",
		25: "FBD2",
		26: "DDR4",
		27: "LPDDR",
		28: "LPDDR2",
		29: "LPDDR3",
		30: "LPDDR4",
		31: "LPDDR5",
	}

	memType := memoryModules[0].MemoryType
	if typeStr, ok := memoryTypes[memType]; ok {
		return typeStr
	}
	return fmt.Sprintf("Unknown (%d)", memType)
}
