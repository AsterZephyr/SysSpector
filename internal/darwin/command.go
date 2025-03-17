package darwin

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// GetSystemInfo collects system information on macOS
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

	// Get model
	modelName, err := runCommand("sysctl", "-n", "hw.model")
	if err != nil {
		log.Printf("Error getting model: %v", err)
	} else {
		info.Model = strings.TrimSpace(modelName)
	}

	// Get serial number
	serialNumber, err := runCommand("ioreg", "-c", "IOPlatformExpertDevice", "-d", "2")
	if err != nil {
		log.Printf("Error getting serial number: %v", err)
	} else {
		serialRegex := regexp.MustCompile(`"IOPlatformSerialNumber" = "([^"]+)"`)
		matches := serialRegex.FindStringSubmatch(serialNumber)
		if len(matches) > 1 {
			info.SerialNumber = matches[1]
		}
	}

	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Printf("Error getting CPU info: %v", err)
	} else if len(cpuInfo) > 0 {
		info.CPU = model.CPUInfo{
			Model: cpuInfo[0].ModelName,
			Cores: len(cpuInfo),
		}
	}

	// Get memory info
	memStats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory info: %v", err)
	} else {
		info.Memory = model.MemoryInfo{
			Total: memStats.Total,
		}
	}

	// Get memory type
	memoryType, err := runCommand("system_profiler", "SPMemoryDataType")
	if err != nil {
		log.Printf("Error getting memory type: %v", err)
	} else {
		typeRegex := regexp.MustCompile(`Type: (DDR\d|LPDDR\d)`)
		matches := typeRegex.FindStringSubmatch(memoryType)
		if len(matches) > 1 {
			info.Memory.Type = matches[1]
		}
	}

	// Get disk info
	diskStats, err := disk.Partitions(false)
	if err != nil {
		log.Printf("Error getting disk partitions: %v", err)
	}

	// Get disk details using diskutil
	_, err = runCommand("diskutil", "list", "-plist")
	if err != nil {
		log.Printf("Error getting disk list: %v", err)
	}

	// Parse disk information
	for _, part := range diskStats {
		if strings.HasPrefix(part.Device, "/dev/disk") {
			diskName := strings.TrimPrefix(part.Device, "/dev/")
			diskInfo, err := runCommand("diskutil", "info", diskName)
			if err != nil {
				log.Printf("Error getting disk info for %s: %v", diskName, err)
				continue
			}

			// Extract disk information using regex
			modelRegex := regexp.MustCompile(`Device / Media Name:\s+(.+)`)
			sizeRegex := regexp.MustCompile(`Disk Size:\s+.*?(\d+)`)
			serialRegex := regexp.MustCompile(`Serial Number:\s+(.+)`)

			modelMatches := modelRegex.FindStringSubmatch(diskInfo)
			sizeMatches := sizeRegex.FindStringSubmatch(diskInfo)
			serialMatches := serialRegex.FindStringSubmatch(diskInfo)

			disk := model.Disk{
				Name: diskName,
			}

			if len(modelMatches) > 1 {
				disk.Model = strings.TrimSpace(modelMatches[1])
			}

			if len(sizeMatches) > 1 {
				size, _ := strconv.ParseUint(sizeMatches[1], 10, 64)
				disk.Size = size
			}

			if len(serialMatches) > 1 {
				disk.Serial = strings.TrimSpace(serialMatches[1])
			}

			info.Disks = append(info.Disks, disk)
		}
	}

	// Get BRUUID
	bruuid, err := runCommand("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err != nil {
		log.Printf("Error getting BRUUID: %v", err)
	} else {
		uuidRegex := regexp.MustCompile(`"IOPlatformUUID" = "([^"]+)"`)
		matches := uuidRegex.FindStringSubmatch(bruuid)
		if len(matches) > 1 {
			info.UUID = matches[1]
		}
	}

	return info, nil
}

// runCommand executes a system command and returns its output
func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute %s: %v", command, err)
	}
	return out.String(), nil
}
