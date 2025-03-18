package darwin

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AsterZephyr/SysSpector/pkg/model"
	"github.com/shirou/gopsutil/v3/process"
	"howett.net/plist"
)

// GetSystemSoftwareInfo 收集macOS系统的系统信息和软件信息
func GetSystemSoftwareInfo(info *model.SystemInfo) error {
	var err error

	// 获取系统版本
	err = getSystemVersion(info)
	if err != nil {
		log.Printf("Error getting system version: %v", err)
	}

	// 获取电脑名称
	err = getComputerName(info)
	if err != nil {
		log.Printf("Error getting computer name: %v", err)
	}

	// 获取启动时间
	err = getUpTime(info)
	if err != nil {
		log.Printf("Error getting up time: %v", err)
	}

	// 获取已安装应用信息
	err = getInstalledApps(info)
	if err != nil {
		log.Printf("Error getting installed apps: %v", err)
	}

	// 获取正在运行的应用信息
	err = getRunningApps(info)
	if err != nil {
		log.Printf("Error getting running apps: %v", err)
	}

	return nil
}

// getSystemVersion 获取系统版本
func getSystemVersion(info *model.SystemInfo) error {
	// 使用sw_vers命令获取系统版本
	output, err := runCommand("sw_vers")
	if err != nil {
		return err
	}

	// 解析系统版本
	versionRegex := regexp.MustCompile(`ProductVersion:\s+(.+)`)
	versionMatches := versionRegex.FindStringSubmatch(output)

	buildRegex := regexp.MustCompile(`BuildVersion:\s+(.+)`)
	buildMatches := buildRegex.FindStringSubmatch(output)

	if len(versionMatches) > 1 && len(buildMatches) > 1 {
		version := strings.TrimSpace(versionMatches[1])
		build := strings.TrimSpace(buildMatches[1])
		info.SystemVersion = fmt.Sprintf("macOS %s (%s)", version, build)
	} else if len(versionMatches) > 1 {
		version := strings.TrimSpace(versionMatches[1])
		info.SystemVersion = fmt.Sprintf("macOS %s", version)
	}

	return nil
}

// getComputerName 获取电脑名称
func getComputerName(info *model.SystemInfo) error {
	// 使用hostname命令获取电脑名称
	output, err := runCommand("hostname")
	if err != nil {
		return err
	}

	// 设置电脑名称
	info.ComputerName = strings.TrimSpace(output)

	return nil
}

// getUpTime 获取启动时间
func getUpTime(info *model.SystemInfo) error {
	// 使用sysctl命令获取启动时间戳
	output, err := runCommand("sysctl", "-n", "kern.boottime")
	if err != nil {
		return err
	}

	// 解析启动时间戳
	secRegex := regexp.MustCompile(`sec = (\d+)`)
	secMatches := secRegex.FindStringSubmatch(output)

	if len(secMatches) > 1 {
		bootTimeSec, _ := strconv.ParseInt(secMatches[1], 10, 64)
		bootTime := time.Unix(bootTimeSec, 0)
		uptime := time.Since(bootTime)

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

	return nil
}

// getInstalledApps 获取已安装应用信息
func getInstalledApps(info *model.SystemInfo) error {
	// 应用程序目录
	appsDir := "/Applications"

	// 遍历应用程序目录
	err := filepath.Walk(appsDir, func(path string, fileInfo fs.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续处理其他文件
		}

		// 只处理.app目录
		if fileInfo.IsDir() && strings.HasSuffix(path, ".app") {
			// 获取应用信息
			appInfo, err := getAppInfo(path)
			if err == nil {
				info.InstalledApps = append(info.InstalledApps, appInfo)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// getAppInfo 获取应用信息
func getAppInfo(appPath string) (model.AppInfo, error) {
	appInfo := model.AppInfo{
		Path: appPath,
	}

	// 获取应用名称（从路径中提取）
	appInfo.Name = filepath.Base(appPath)
	appInfo.Name = strings.TrimSuffix(appInfo.Name, ".app")

	// 获取应用安装日期（使用目录修改时间）
	fileInfo, err := os.Stat(appPath)
	if err == nil {
		appInfo.InstallDate = fileInfo.ModTime().Format("2006年01月02日 15:04:05")
	}

	// 获取应用版本（从Info.plist文件中提取）
	infoPlistPath := filepath.Join(appPath, "Contents", "Info.plist")
	if _, err := os.Stat(infoPlistPath); err == nil {
		// 读取plist文件
		plistFile, err := os.Open(infoPlistPath)
		if err == nil {
			defer plistFile.Close()

			// 解析plist文件
			var plistData map[string]interface{}
			decoder := plist.NewDecoder(plistFile)
			err = decoder.Decode(&plistData)
			if err == nil {
				// 获取版本信息
				if version, ok := plistData["CFBundleShortVersionString"].(string); ok {
					appInfo.Version = version
				} else if version, ok := plistData["CFBundleVersion"].(string); ok {
					appInfo.Version = version
				}
			}
		}
	}

	return appInfo, nil
}

// getRunningApps 获取正在运行的应用信息
func getRunningApps(info *model.SystemInfo) error {
	// 使用gopsutil获取进程列表
	processes, err := process.Processes()
	if err != nil {
		return err
	}

	// 处理每个进程
	for _, p := range processes {
		// 获取进程名称
		name, err := p.Name()
		if err != nil {
			continue
		}

		// 跳过系统进程
		if strings.HasPrefix(name, "com.apple.") || strings.HasPrefix(name, "system") {
			continue
		}

		// 获取进程CPU使用率
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			cpuPercent = 0
		}

		// 获取进程内存使用量
		memInfo, err := p.MemoryInfo()
		var memUsage uint64 = 0
		if err == nil && memInfo != nil {
			memUsage = memInfo.RSS
		}

		// 创建进程信息
		processInfo := model.ProcessInfo{
			PID:    int(p.Pid),
			Name:   name,
			CPU:    cpuPercent,
			Memory: memUsage,
		}

		// 添加到运行应用列表
		info.RunningApps = append(info.RunningApps, processInfo)
	}

	return nil
}
