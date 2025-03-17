package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/AsterZephyr/SysSpector/internal/darwin"
	"github.com/AsterZephyr/SysSpector/internal/windows"
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

func main() {
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
			log.Fatalf("Error collecting Windows system information: %v", err)
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
	printFormattedInfo(sysInfo)

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
}

// printFormattedInfo 打印格式化的系统信息
func printFormattedInfo(info model.SystemInfo) {
	fmt.Println(formatSystemInfo(info))
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

	// 3. 设备型号（与型号名称相同）
	if info.Model != "" {
		sb.WriteString(fmt.Sprintf("3. 型号标识符：%s\n", info.Model))
	} else {
		sb.WriteString("\n")
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
		} else {
			// 其他处理器格式：Apple M3 Pro X
			cpuDesc = fmt.Sprintf("%s %d", cpuDesc, info.CPU.Cores)
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
