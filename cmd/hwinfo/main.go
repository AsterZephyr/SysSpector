package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	fmt.Println("开始采集硬件信息...")

	// 采集网卡信息
	fmt.Println("\n===== 网卡信息 =====")
	getNetworkCardInfo()

	// 采集显卡信息
	fmt.Println("\n===== 显卡信息 =====")
	getGraphicsCardInfo()

	// 采集显示器信息
	fmt.Println("\n===== 显示器信息 =====")
	getDisplayInfo()
}

// 获取网卡信息
func getNetworkCardInfo() {
	fmt.Println("正在采集网卡信息...")

	// 使用ifconfig命令获取网络接口信息
	output, err := exec.Command("ifconfig", "-a").Output()
	if err != nil {
		fmt.Printf("获取网卡信息失败: %v\n", err)
		return
	}

	// 解析输出
	interfaces := strings.Split(string(output), "\n\n")
	count := 0
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

		// 直接输出网卡信息
		fmt.Printf("网卡 #%d:\n", count+1)
		fmt.Printf("  名称: %s\n", name)
		fmt.Printf("  MAC地址: %s\n", mac)
		if len(ipAddresses) > 0 {
			fmt.Printf("  IP地址: %s\n", strings.Join(ipAddresses, ", "))
		}
		fmt.Println()

		count++
	}

	fmt.Printf("共采集到 %d 个网卡信息\n", count)
}

// 获取显卡信息
func getGraphicsCardInfo() {
	fmt.Println("正在采集显卡信息...")

	// 使用system_profiler命令获取显卡信息
	output, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		fmt.Printf("获取显卡信息失败: %v\n", err)
		return
	}

	// 解析输出
	gpuRegex := regexp.MustCompile(`(?s)Chipset Model: (.+?)\n.*?Type: (.+?)\n`)
	gpuMatches := gpuRegex.FindAllStringSubmatch(string(output), -1)

	count := 0
	for _, match := range gpuMatches {
		if len(match) < 3 {
			continue
		}

		model := strings.TrimSpace(match[1])
		gpuType := strings.TrimSpace(match[2])

		// 直接输出显卡信息
		fmt.Printf("显卡 #%d:\n", count+1)
		fmt.Printf("  型号: %s\n", model)
		fmt.Printf("  类型: %s\n", gpuType)
		fmt.Println()

		count++
	}

	fmt.Printf("共采集到 %d 个显卡信息\n", count)
}

// 获取显示器信息
func getDisplayInfo() {
	fmt.Println("正在采集显示器信息...")

	// 使用system_profiler命令获取显示器信息
	output, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		fmt.Printf("获取显示器信息失败: %v\n", err)
		return
	}

	// 解析输出
	// 查找显示器部分
	displayRegex := regexp.MustCompile(`(?s)Display Type: (.+?)\n.*?Resolution: (.+?)\n`)
	displayMatches := displayRegex.FindAllStringSubmatch(string(output), -1)

	count := 0
	for _, match := range displayMatches {
		if len(match) < 3 {
			continue
		}

		displayType := strings.TrimSpace(match[1])
		resolution := strings.TrimSpace(match[2])

		// 直接输出显示器信息
		fmt.Printf("显示器 #%d:\n", count+1)
		fmt.Printf("  类型: %s\n", displayType)
		fmt.Printf("  分辨率: %s\n", resolution)
		fmt.Println()

		count++
	}

	fmt.Printf("共采集到 %d 个显示器信息\n", count)
}
