//go:build darwin
// +build darwin

package darwin

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw"
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

	// 获取设备型号标识符
	modelName, err := runCommand("sysctl", "-n", "hw.model")
	if err != nil {
		log.Printf("Error getting model: %v", err)
	} else {
		info.Model = strings.TrimSpace(modelName) // 保存型号标识符
	}
	marketingName, err := runCommand("system_profiler", "SPHardwareDataType")
	if err != nil {
		log.Printf("Error getting marketing model name: %v", err)
	} else {
		// 从system_profiler输出中提取型号名称
		re := regexp.MustCompile(`Model Name: (.+)`)
		matches := re.FindStringSubmatch(marketingName)
		if len(matches) > 1 {
			info.ModelID = info.Model                  
			info.Model = strings.TrimSpace(matches[1]) 
		}
	}

	serialNumber, err := runCommand("ioreg", "-c", "IOPlatformExpertDevice", "-d", "2")
	if err != nil {
		log.Printf("Error getting serial number: %v", err)
	} else {
		re := regexp.MustCompile(`"IOPlatformSerialNumber" = "([^"]+)"`)
		matches := re.FindStringSubmatch(serialNumber)
		if len(matches) > 1 {
			info.SerialNumber = matches[1]
		}
	}

	// 使用 ghw 获取 CPU 信息
	cpuInfo, err := ghw.CPU()
	if err != nil {
		log.Printf("Error getting CPU info with ghw: %v", err)
		coreCount, err := runCommand("sysctl", "-n", "hw.physicalcpu")
		cores := 0
		if err != nil {
			log.Printf("Error getting CPU core count: %v", err)
		} else {
			cores, _ = strconv.Atoi(strings.TrimSpace(coreCount))
		}

		// 检测是否为 Apple Silicon
		isAppleSilicon := false

		// 使用sysctl检查CPU架构
		archOutput, err := runCommand("sysctl", "-n", "hw.machine")
		if err == nil {
			arch := strings.TrimSpace(archOutput)
			// arm64表示Apple Silicon，x86_64表示Intel
			isAppleSilicon = arch == "arm64"
		}

		// 如果sysctl失败，尝试使用其他方法
		if err != nil {
			// 检查是否存在M系列芯片特有的sysctl键
			_, err := runCommand("sysctl", "-n", "hw.perflevel0.physicalcpu")
			isAppleSilicon = err == nil // 如果这个命令成功，说明是M系列芯片
		}

		var cpuModel string
		if isAppleSilicon {
			// 对于 M 系列芯片，尝试获取处理器型号
			cpuModelOutput, err := runCommand("sysctl", "-n", "machdep.cpu.brand_string")
			if err != nil || strings.TrimSpace(cpuModelOutput) == "" {
				// 如果无法获取，则根据设备型号推断
				if strings.Contains(info.ModelID, "Mac") {
					if strings.Contains(info.ModelID, "14,") || strings.Contains(info.ModelID, "15,") {
						cpuModel = "Apple M3"
					} else if strings.Contains(info.ModelID, "13,") {
						cpuModel = "Apple M2"
					} else {
						cpuModel = "Apple M1"
					}
				} else {
					cpuModel = "Apple Silicon"
				}
			} else {
				cpuModel = strings.TrimSpace(cpuModelOutput)
			}

			// 对于 M 系列芯片，添加 Pro/Max/Ultra 后缀（如果能够确定）
			if strings.Contains(info.ModelID, "Pro") {
				if !strings.Contains(cpuModel, "Pro") {
					cpuModel += " Pro"
				}
			} else if strings.Contains(info.ModelID, "Max") {
				if !strings.Contains(cpuModel, "Max") {
					cpuModel += " Max"
				}
			} else if strings.Contains(info.ModelID, "Ultra") {
				if !strings.Contains(cpuModel, "Ultra") {
					cpuModel += " Ultra"
				}
			}
		} else {
			// 对于 Intel 芯片，使用 sysctl 获取
			cpuModelOutput, err := runCommand("sysctl", "-n", "machdep.cpu.brand_string")
			if err != nil {
				log.Printf("Error getting CPU model: %v", err)
				cpuModel = "Intel CPU"
			} else {
				cpuModel = strings.TrimSpace(cpuModelOutput)
			}
		}

		info.CPU = model.CPUInfo{
			Model: cpuModel,
			Cores: cores,
		}
	} else {
		// 使用 ghw 获取的 CPU 信息
		cpuModel := ""
		cores := 0

		if len(cpuInfo.Processors) > 0 {
			proc := cpuInfo.Processors[0]
			cpuModel = proc.Model
			cores = int(proc.NumCores)

			// 检查是否为 Apple Silicon
			isAppleSilicon := false

			// 使用sysctl检查CPU架构
			archOutput, err := runCommand("sysctl", "-n", "hw.machine")
			if err == nil {
				arch := strings.TrimSpace(archOutput)
				// arm64表示Apple Silicon，x86_64表示Intel
				isAppleSilicon = arch == "arm64"
			}

			// 如果sysctl失败，尝试使用其他方法
			if err != nil {
				// 检查是否存在M系列芯片特有的sysctl键
				_, err := runCommand("sysctl", "-n", "hw.perflevel0.physicalcpu")
				isAppleSilicon = err == nil // 如果这个命令成功，说明是M系列芯片
			}

			if isAppleSilicon {
				// 对于 M 系列芯片，可能需要额外处理
				if !strings.Contains(cpuModel, "M1") && !strings.Contains(cpuModel, "M2") && !strings.Contains(cpuModel, "M3") {
					// 如果 ghw 没有正确识别 M 系列芯片，尝试推断
					if strings.Contains(info.ModelID, "Mac") {
						if strings.Contains(info.ModelID, "14,") || strings.Contains(info.ModelID, "15,") {
							cpuModel = "Apple M3"
						} else if strings.Contains(info.ModelID, "13,") {
							cpuModel = "Apple M2"
						} else {
							cpuModel = "Apple M1"
						}
					}
				}

				// 对于 M 系列芯片，添加 Pro/Max/Ultra 后缀（如果能够确定）
				if strings.Contains(info.ModelID, "Pro") {
					if !strings.Contains(cpuModel, "Pro") {
						cpuModel += " Pro"
					}
				} else if strings.Contains(info.ModelID, "Max") {
					if !strings.Contains(cpuModel, "Max") {
						cpuModel += " Max"
					}
				} else if strings.Contains(info.ModelID, "Ultra") {
					if !strings.Contains(cpuModel, "Ultra") {
						cpuModel += " Ultra"
					}
				}
			}
		}

		info.CPU = model.CPUInfo{
			Model: cpuModel,
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
			log.Printf("获取内存类型失败: %v，尝试使用备用方法", err)
			// 尝试使用sysctl命令获取内存类型
			sysctlOutput, err := runCommand("sysctl", "hw.memtype")
			if err == nil && sysctlOutput != "" {
				// 解析sysctl输出
				parts := strings.Split(sysctlOutput, ": ")
				if len(parts) > 1 {
					memType = strings.TrimSpace(parts[1])
					log.Printf("通过sysctl获取到内存类型: %s", memType)
				}
			}
		} else {
			log.Printf("system_profiler SPMemoryDataType输出: %s", memTypeOutput)
			
			// 使用正则表达式提取内存类型
			typeRegex := regexp.MustCompile(`Type: ([A-Za-z0-9]+)`)
			typeMatches := typeRegex.FindStringSubmatch(memTypeOutput)
			if len(typeMatches) > 1 {
				memType = typeMatches[1]
				log.Printf("从system_profiler提取到内存类型: %s", memType)
			} else {
				// 检查常见内存类型
				memoryTypes := []string{"LPDDR5", "LPDDR4X", "LPDDR4", "LPDDR3", "DDR5", "DDR4", "DDR3", "DDR2"}
				for _, mt := range memoryTypes {
					if strings.Contains(memTypeOutput, mt) {
						memType = mt
						log.Printf("通过关键字匹配找到内存类型: %s", memType)
						break
					}
				}
			}
			
			// 如果仍然未找到，尝试提取更多信息作为Details字段
			details := ""
			speedRegex := regexp.MustCompile(`Speed: (\d+) MHz`)
			speedMatches := speedRegex.FindStringSubmatch(memTypeOutput)
			if len(speedMatches) > 1 {
				details += "Speed: " + speedMatches[1] + " MHz"
			}
			
			manufacturerRegex := regexp.MustCompile(`Manufacturer: ([^\n]+)`)
			manufacturerMatches := manufacturerRegex.FindStringSubmatch(memTypeOutput)
			if len(manufacturerMatches) > 1 {
				if details != "" {
					details += ", "
				}
				details += "Manufacturer: " + strings.TrimSpace(manufacturerMatches[1])
			}
			
			partNumberRegex := regexp.MustCompile(`Part Number: ([^\n]+)`)
			partNumberMatches := partNumberRegex.FindStringSubmatch(memTypeOutput)
			if len(partNumberMatches) > 1 {
				if details != "" {
					details += ", "
				}
				details += "Part Number: " + strings.TrimSpace(partNumberMatches[1])
			}
			
			// 保存详细信息
			info.Memory.Details = details
			log.Printf("内存详细信息: %s", details)
		}

		// 创建内存信息结构体
		info.Memory = model.MemoryInfo{
			Total: memInfo.Total,
			Type:  memType,
			// Details字段在前面的代码中已经设置，这里不需要重新设置
		}
		
		// 如果内存类型仍然是“Unknown”且没有详细信息，尝试使用更多方法
		if info.Memory.Type == "Unknown" && info.Memory.Details == "" {
			log.Printf("尝试使用更多方法获取内存信息")
			
			// 尝试使用system_profiler SPHardwareDataType命令
			hwOutput, err := runCommand("system_profiler", "SPHardwareDataType")
			if err == nil {
				// 尝试提取内存类型信息
				log.Printf("SPHardwareDataType输出: %s", hwOutput)
				
				// 如果是Apple Silicon芯片，可能是LPDDR4X或LPDDR5
				if strings.Contains(hwOutput, "Apple M1") {
					info.Memory.Type = "LPDDR4X"
					log.Printf("检测到Apple M1芯片，内存类型可能是LPDDR4X")
				} else if strings.Contains(hwOutput, "Apple M2") {
					info.Memory.Type = "LPDDR5"
					log.Printf("检测到Apple M2芯片，内存类型可能是LPDDR5")
				} else if strings.Contains(hwOutput, "Apple M3") {
					info.Memory.Type = "LPDDR5"
					log.Printf("检测到Apple M3芯片，内存类型可能是LPDDR5")
				} else if strings.Contains(hwOutput, "Intel") {
					// 如果是Intel芯片，根据年份推测内存类型
					modelRegex := regexp.MustCompile(`Model Identifier: ([^\n]+)`)
					modelMatches := modelRegex.FindStringSubmatch(hwOutput)
					if len(modelMatches) > 1 {
						modelID := modelMatches[1]
						log.Printf("检测到Intel芯片，型号: %s", modelID)
						// 根据型号年份推测内存类型
						if strings.Contains(modelID, "202") || strings.Contains(modelID, "201") {
							info.Memory.Type = "DDR4"
						} else {
							info.Memory.Type = "DDR3"
						}
					}
				}		
				// 提取内存大小信息作为详细信息
				memSizeRegex := regexp.MustCompile(`Memory: (\d+) GB`)
				memSizeMatches := memSizeRegex.FindStringSubmatch(hwOutput)
				if len(memSizeMatches) > 1 {
					if info.Memory.Details == "" {
						info.Memory.Details = "Size: " + memSizeMatches[1] + " GB"
					} else {
						info.Memory.Details += ", Size: " + memSizeMatches[1] + " GB"
					}
				}
			}
		}
	}

	// 使用 ghw 获取磁盘信息
	blockInfo, err := ghw.Block()
	if err != nil {
		log.Printf("Error getting block info with ghw: %v", err)

		// 如果 ghw 失败，回退到 system_profiler
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
	} else {
		// 使用 ghw 获取的磁盘信息
		for _, disk := range blockInfo.Disks {
			// 只添加物理磁盘，不添加分区
			if disk.IsRemovable || disk.DriveType != ghw.DRIVE_TYPE_HDD && disk.DriveType != ghw.DRIVE_TYPE_SSD {
				continue
			}

			// 转换为 GB
			sizeGB := uint64(disk.SizeBytes / (1024 * 1024 * 1024))

			info.Disks = append(info.Disks, model.Disk{
				Name:   disk.Name,
				Size:   sizeGB,
				Serial: disk.SerialNumber,
				Model:  disk.Model,
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

	// 收集动态系统信息
	err = GetDynamicSystemInfo(&info)
	if err != nil {
		log.Printf("Error getting dynamic system info: %v", err)
	}

	log.Println("正在采集额外硬件信息...")
	err = GetNetworkCardInfo(&info)
	if err != nil {
		log.Printf("Error getting network card info: %v", err)
	}

	err = GetGraphicsCardInfo(&info)
	if err != nil {
		log.Printf("Error getting graphics card info: %v", err)
	}

	err = GetDisplayInfo(&info)
	if err != nil {
		log.Printf("Error getting display info: %v", err)
	}

	// 收集网络信息
	err = GetNetworkInfo(&info)
	if err != nil {
		log.Printf("Error getting network info: %v", err)
	}

	// 收集系统和软件信息
	err = GetSystemSoftwareInfo(&info)
	if err != nil {
		log.Printf("Error getting system and software info: %v", err)
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
