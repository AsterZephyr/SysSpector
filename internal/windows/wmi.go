//go:build windows
// +build windows

package windows

import (
	"fmt"
	"log"
	"strconv"

	"github.com/StackExchange/wmi"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// WMI query structures
type win32ComputerSystem struct {
	Model               string
	Name                string
	TotalPhysicalMemory uint64
}

type win32Processor struct {
	Name          string
	NumberOfCores uint32
}

type win32BIOS struct {
	SerialNumber string
}

type win32PhysicalMemory struct {
	Capacity   uint64
	MemoryType uint16
}

type win32DiskDrive struct {
	Caption      string
	Model        string
	Size         string
	SerialNumber string
}

type win32ComputerSystemProduct struct {
	UUID string
}

// GetSystemInfo collects system information on Windows
func GetSystemInfo() (model.SystemInfo, error) {
	var info model.SystemInfo
	var err error

	// Get hostname and OS info
	hostInfo, err := host.Info()
	if err != nil {
		log.Printf("Error getting host info: %v", err)
	} else {
		info.Hostname = hostInfo.Hostname
		info.OS = hostInfo.Platform + " " + hostInfo.PlatformVersion
	}

	// Get computer system info
	var computerSystems []win32ComputerSystem
	err = wmi.Query("SELECT Model, Name, TotalPhysicalMemory FROM Win32_ComputerSystem", &computerSystems)
	if err != nil {
		log.Printf("Error querying Win32_ComputerSystem: %v", err)
	} else if len(computerSystems) > 0 {
		info.Model = computerSystems[0].Model
	}

	// Get serial number
	var biosInfo []win32BIOS
	err = wmi.Query("SELECT SerialNumber FROM Win32_BIOS", &biosInfo)
	if err != nil {
		log.Printf("Error querying Win32_BIOS: %v", err)
	} else if len(biosInfo) > 0 {
		info.SerialNumber = biosInfo[0].SerialNumber
	}

	// Get CPU info
	var processors []win32Processor
	err = wmi.Query("SELECT Name, NumberOfCores FROM Win32_Processor", &processors)
	if err != nil {
		log.Printf("Error querying Win32_Processor: %v", err)
	} else if len(processors) > 0 {
		info.CPU = model.CPUInfo{
			Model: processors[0].Name,
			Cores: int(processors[0].NumberOfCores),
		}
	}

	// Get memory info
	var memoryInfo []win32PhysicalMemory
	err = wmi.Query("SELECT Capacity, MemoryType FROM Win32_PhysicalMemory", &memoryInfo)
	if err != nil {
		log.Printf("Error querying Win32_PhysicalMemory: %v", err)
	}

	// Calculate total memory and determine type
	memStats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory info: %v", err)
	} else {
		info.Memory = model.MemoryInfo{
			Total: memStats.Total,
			Type:  getMemoryTypeString(memoryInfo),
		}
	}

	// Get disk info
	var diskDrives []win32DiskDrive
	err = wmi.Query("SELECT Caption, Model, Size, SerialNumber FROM Win32_DiskDrive", &diskDrives)
	if err != nil {
		log.Printf("Error querying Win32_DiskDrive: %v", err)
	} else {
		for _, d := range diskDrives {
			size, _ := strconv.ParseUint(d.Size, 10, 64)
			info.Disks = append(info.Disks, model.Disk{
				Name:   d.Caption,
				Model:  d.Model,
				Size:   size,
				Serial: d.SerialNumber,
			})
		}
	}

	// Get UUID
	var systemProducts []win32ComputerSystemProduct
	err = wmi.Query("SELECT UUID FROM Win32_ComputerSystemProduct", &systemProducts)
	if err != nil {
		log.Printf("Error querying Win32_ComputerSystemProduct: %v", err)
	} else if len(systemProducts) > 0 {
		info.UUID = systemProducts[0].UUID
	}

	return info, nil
}

// getMemoryTypeString converts WMI memory type code to string description
func getMemoryTypeString(memoryModules []win32PhysicalMemory) string {
	if len(memoryModules) == 0 {
		return "Unknown"
	}

	// Memory type codes according to WMI documentation
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
