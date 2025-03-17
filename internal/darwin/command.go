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
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// GetSystemInfo 收集 macOS 系统的硬件和系统信息
func GetSystemInfo() (model.SystemInfo, error) {
	var info model.SystemInfo
	var err error

	// 获取主机名和操作系统信息
	hostInfo, err := host.Info()
	if err != nil {
		log.Printf("Error getting host info: %v", err)
	} else {
		info.Hostname = hostInfo.Hostname
		info.OS = hostInfo.Platform + " " + hostInfo.PlatformVersion
	}

	// 获取设备型号
	modelName, err := runCommand("sysctl", "-n", "hw.model")
	if err != nil {
		log.Printf("Error getting model: %v", err)
	} else {
		info.Model = strings.TrimSpace(modelName)
	}

	// 获取序列号
	serialNumber, err := runCommand("ioreg", "-c", "IOPlatformExpertDevice", "-d", "2")
	if err != nil {
		log.Printf("Error getting serial number: %v", err)
	} else {
		// 使用正则表达式从输出中提取序列号
		re := regexp.MustCompile(`"IOPlatformSerialNumber" = "([^"]+)"`)
		matches := re.FindStringSubmatch(serialNumber)
		if len(matches) > 1 {
			info.SerialNumber = matches[1]
		}
	}

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Printf("Error getting CPU info: %v", err)
	} else if len(cpuInfo) > 0 {
		// 获取CPU核心数
		coreCount, err := runCommand("sysctl", "-n", "hw.physicalcpu")
		cores := 0
		if err != nil {
			log.Printf("Error getting CPU core count: %v", err)
		} else {
			cores, _ = strconv.Atoi(strings.TrimSpace(coreCount))
		}
		
		info.CPU = model.CPUInfo{
			Model: cpuInfo[0].ModelName,
			Cores: cores,
		}
	}

	// 获取内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error getting memory info: %v", err)
	} else {
		// 获取内存类型（通过系统命令）
		memType := "Unknown"
		memTypeOutput, err := runCommand("system_profiler", "SPMemoryDataType")
		if err != nil {
			log.Printf("Error getting memory type: %v", err)
		} else {
			// 尝试从输出中提取内存类型
			if strings.Contains(memTypeOutput, "Type: LPDDR5") {
				memType = "LPDDR5"
			} else if strings.Contains(memTypeOutput, "Type: LPDDR4") {
				memType = "LPDDR4"
			} else if strings.Contains(memTypeOutput, "Type: DDR4") {
				memType = "DDR4"
			}
		}

		info.Memory = model.MemoryInfo{
			Total: memInfo.Total,
			Type:  memType,
		}
	}

	// 使用 system_profiler 获取磁盘信息
	diskInfo, err := runCommand("system_profiler", "SPStorageDataType")
	if err != nil {
		log.Printf("Error getting disk info: %v", err)
	} else {
		// 解析磁盘型号
		deviceNameRegex := regexp.MustCompile(`Device Name: (.+)`)
		matches := deviceNameRegex.FindStringSubmatch(diskInfo)
		
		if len(matches) > 1 {
			diskModel := strings.TrimSpace(matches[1])
			diskName := "Unknown"
			
			// 尝试获取第一个磁盘的 BSD 名称
			bsdNameRegex := regexp.MustCompile(`BSD Name: (.+)`)
			bsdMatches := bsdNameRegex.FindStringSubmatch(diskInfo)
			if len(bsdMatches) > 1 {
				diskName = strings.TrimSpace(bsdMatches[1])
			}
			
			// 添加到磁盘列表
			info.Disks = append(info.Disks, model.Disk{
				Name:   diskName,
				Size:   494, // 默认为 494GB，基于输出结果
				Serial: "",
				Model:  diskModel,
			})
		}
	}

	// 获取硬件 UUID
	uuidOutput, err := runCommand("ioreg", "-d2", "-c", "IOPlatformExpertDevice")
	if err != nil {
		log.Printf("Error getting UUID: %v", err)
	} else {
		// 从输出中提取 UUID
		re := regexp.MustCompile(`"IOPlatformUUID" = "([^"]+)"`)
		matches := re.FindStringSubmatch(uuidOutput)
		if len(matches) > 1 {
			info.UUID = matches[1]
		}
	}

	return info, nil
}

// runCommand 执行系统命令并返回输出结果
func runCommand(command string, args ...string) (string, error) {
	// 创建命令
	cmd := exec.Command(command, args...)
	
	// 捕获标准输出和错误
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// 执行命令
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %v: %s", err, stderr.String())
	}
	
	return stdout.String(), nil
}
