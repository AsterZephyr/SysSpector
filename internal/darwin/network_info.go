package darwin

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
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

	// 获取IP地址和MAC地址
	err = getIPAndMACAddresses(&networkInfo)
	if err != nil {
		log.Printf("Error getting IP and MAC addresses: %v", err)
	}

	// 获取AWDL状态
	err = getAWDLStatus(&networkInfo)
	if err != nil {
		log.Printf("Error getting AWDL status: %v", err)
	}

	// 获取网络接口信息
	err = getNetworkInterfaces(&networkInfo)
	if err != nil {
		log.Printf("Error getting network interfaces: %v", err)
	}

	// 获取网络流量信息
	err = getNetworkTraffic(&networkInfo)
	if err != nil {
		log.Printf("Error getting network traffic: %v", err)
	}

	// 获取网络延迟信息
	err = getNetworkLatency(&networkInfo)
	if err != nil {
		log.Printf("Error getting network latency: %v", err)
	}

	// 获取VPN信息
	err = getVPNInfo(&networkInfo)
	if err != nil {
		log.Printf("Error getting VPN info: %v", err)
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

	// 获取网络代理状态
	err = getProxyStatus(&networkInfo)
	if err != nil {
		log.Printf("Error getting proxy status: %v", err)
	}

	// 将收集到的网络信息设置到系统信息中
	info.Network = networkInfo

	return nil
}

// getWiFiInfo 获取WiFi信息
func getWiFiInfo(info *model.NetworkInfo) error {
	// 使用airport命令获取WiFi信息
	output, err := runCommand("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-I")
	if err != nil {
		return err
	}

	// 初始化WiFi信息
	wifiInfo := model.WiFiInfo{}

	// 解析WiFi信息
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// 解析SSID
		if strings.Contains(line, " SSID: ") {
			parts := strings.Split(line, "SSID: ")
			if len(parts) > 1 {
				wifiInfo.SSID = strings.TrimSpace(parts[1])
				wifiInfo.IsConnected = true
			}
		}

		// 解析BSSID
		if strings.Contains(line, " BSSID: ") {
			parts := strings.Split(line, "BSSID: ")
			if len(parts) > 1 {
				wifiInfo.BSSID = strings.TrimSpace(parts[1])
			}
		}

		// 解析信号强度
		if strings.Contains(line, "agrCtlRSSI: ") {
			parts := strings.Split(line, "agrCtlRSSI: ")
			if len(parts) > 1 {
				signalStrength, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.SignalStrength = signalStrength
				}
			}
		}

		// 解析频道
		if strings.Contains(line, "channel: ") {
			parts := strings.Split(line, "channel: ")
			if len(parts) > 1 {
				channel, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.Channel = channel
				}
			}
		}

		// 解析频率
		if strings.Contains(line, "frequency: ") {
			parts := strings.Split(line, "frequency: ")
			if len(parts) > 1 {
				freqStr := strings.TrimSpace(parts[1])
				if strings.HasSuffix(freqStr, " GHz") {
					freqStr = strings.TrimSuffix(freqStr, " GHz")
					freq, err := strconv.ParseFloat(freqStr, 64)
					if err == nil {
						wifiInfo.Frequency = freq
					}
				}
			}
		}

		// 解析传输速率
		if strings.Contains(line, "lastTxRate: ") {
			parts := strings.Split(line, "lastTxRate: ")
			if len(parts) > 1 {
				txRate, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.TxRate = txRate
				}
			}
		}
	}

	// 设置WiFi信息
	info.WiFi = wifiInfo

	return nil
}

// getIPAndMACAddresses 获取IP地址和MAC地址
func getIPAndMACAddresses(info *model.NetworkInfo) error {
	// 使用ifconfig命令获取网络接口信息
	output, err := runCommand("ifconfig")
	if err != nil {
		return err
	}

	// 解析网络接口信息
	interfaces := make(map[string]model.NetInterfaceInfo)

	// 当前处理的接口名称
	var currentIface string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// 检查是否是新接口的开始
		ifaceRegex := regexp.MustCompile(`^([a-zA-Z0-9]+):`)
		ifaceMatches := ifaceRegex.FindStringSubmatch(line)

		if len(ifaceMatches) > 1 {
			currentIface = ifaceMatches[1]
			// 初始化新接口
			iface := model.NetInterfaceInfo{
				Name: currentIface,
			}
			interfaces[currentIface] = iface
		}

		// 如果当前有处理中的接口，解析其信息
		if currentIface != "" {
			// 解析MAC地址
			macRegex := regexp.MustCompile(`ether\s+([0-9a-f:]+)`)
			macMatches := macRegex.FindStringSubmatch(line)
			if len(macMatches) > 1 {
				iface := interfaces[currentIface]
				iface.MacAddress = macMatches[1]
				interfaces[currentIface] = iface
			}

			// 解析IPv4地址
			ipv4Regex := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
			ipv4Matches := ipv4Regex.FindStringSubmatch(line)
			if len(ipv4Matches) > 1 {
				iface := interfaces[currentIface]
				iface.IP = ipv4Matches[1]
				interfaces[currentIface] = iface
			}

			// 解析IPv6地址
			ipv6Regex := regexp.MustCompile(`inet6\s+([0-9a-f:]+)`)
			ipv6Matches := ipv6Regex.FindStringSubmatch(line)
			if len(ipv6Matches) > 1 {
				iface := interfaces[currentIface]
				iface.IP = ipv6Matches[1]
				interfaces[currentIface] = iface
			}

			// 解析状态
			statusRegex := regexp.MustCompile(`status:\s+(.+)`)
			statusMatches := statusRegex.FindStringSubmatch(line)
			if len(statusMatches) > 1 {
				iface := interfaces[currentIface]
				iface.IsUp = statusMatches[1] == "active"
				interfaces[currentIface] = iface
			}
		}
	}

	// 将解析的接口信息添加到网络信息中
	for _, iface := range interfaces {
		// 跳过回环接口和没有IP地址的接口
		if iface.Name == "lo0" || iface.IP == "" {
			continue
		}

		info.Interfaces = append(info.Interfaces, iface)
	}

	return nil
}

// getAWDLStatus 获取AWDL状态
func getAWDLStatus(info *model.NetworkInfo) error {
	// 使用ifconfig命令获取awdl0接口的状态
	output, err := runCommand("ifconfig", "awdl0")
	if err != nil {
		// 如果命令失败，可能是因为awdl0接口不存在
		return nil
	}

	// 检查AWDL是否启用
	if strings.Contains(output, "UP") {
		info.AWDLEnabled = true
	} else {
		info.AWDLEnabled = false
	}

	return nil
}

// getNetworkInterfaces 获取网络接口信息
func getNetworkInterfaces(info *model.NetworkInfo) error {
	// 使用Go标准库获取网络接口信息
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	// 处理每个网络接口
	for _, iface := range ifaces {
		// 跳过回环接口和没有UP标志的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取接口的地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 如果接口没有地址，跳过
		if len(addrs) == 0 {
			continue
		}

		// 创建网络接口信息
		netIface := model.NetInterfaceInfo{
			Name:       iface.Name,
			MacAddress: iface.HardwareAddr.String(),
			IsUp:       true,
			IP:         addrs[0].String(),
		}

		// 添加到接口列表
		info.Interfaces = append(info.Interfaces, netIface)
	}

	return nil
}

// getNetworkTraffic 获取网络流量信息
func getNetworkTraffic(info *model.NetworkInfo) error {
	// 使用netstat命令获取网络流量信息
	output, err := runCommand("netstat", "-ib")
	if err != nil {
		return err
	}

	// 解析网络流量信息
	scanner := bufio.NewScanner(strings.NewReader(output))

	// 跳过标题行
	scanner.Scan()

	// 处理每一行
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// 确保有足够的字段
		if len(fields) < 10 {
			continue
		}

		// 获取接口名称
		ifaceName := fields[0]

		// 跳过回环接口
		if ifaceName == "lo0" {
			continue
		}

		// 查找对应的接口
		for i, iface := range info.Interfaces {
			if iface.Name == ifaceName {
				// 解析接收和发送的字节数
				inBytes, err1 := strconv.ParseUint(fields[6], 10, 64)
				outBytes, err2 := strconv.ParseUint(fields[9], 10, 64)

				if err1 == nil && err2 == nil {
					// 更新接口的流量信息
					info.Interfaces[i].InBytes = inBytes
					info.Interfaces[i].OutBytes = outBytes
				}

				break
			}
		}
	}

	return nil
}

// getNetworkLatency 获取网络延迟信息
func getNetworkLatency(info *model.NetworkInfo) error {
	// 使用ping命令获取网络延迟信息
	output, err := runCommand("ping", "-c", "3", "-q", "8.8.8.8")
	if err != nil {
		return err
	}

	// 解析ping结果
	latencyRegex := regexp.MustCompile(`min/avg/max/stddev = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+)`)
	matches := latencyRegex.FindStringSubmatch(output)

	if len(matches) > 2 {
		// 解析平均延迟
		avgLatency, err := strconv.ParseFloat(matches[2], 64)
		if err == nil {
			info.AvgLatency = avgLatency
		}
	}

	return nil
}

// getVPNInfo 获取VPN信息
func getVPNInfo(info *model.NetworkInfo) error {
	// 使用networksetup命令获取VPN信息
	output, err := runCommand("networksetup", "-listallnetworkservices")
	if err != nil {
		return err
	}

	// 检查是否有VPN服务
	vpnServices := []string{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "VPN") {
			vpnServices = append(vpnServices, strings.TrimSpace(line))
		}
	}

	// 如果有VPN服务，检查其状态
	if len(vpnServices) > 0 {
		info.VPN = model.VPNInfo{
			IsConnected: false,
			Services:    vpnServices,
		}

		// 检查是否有VPN连接
		ifconfigOutput, err := runCommand("ifconfig")
		if err == nil {
			// 检查是否有utun接口（通常用于VPN）
			if strings.Contains(ifconfigOutput, "utun") {
				info.VPN.IsConnected = true
			}
		}
	}

	return nil
}

// getDNSConfig 获取DNS配置
func getDNSConfig(info *model.NetworkInfo) error {
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
			for _, server := range info.DNSServers {
				if server == dnsServer {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				info.DNSServers = append(info.DNSServers, dnsServer)
			}
		}
	}

	return nil
}

// getPublicIP 获取公网IP
func getPublicIP(info *model.NetworkInfo) error {
	// 使用外部服务获取公网IP
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://api.ipify.org?format=json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析JSON响应
	var result struct {
		IP string `json:"ip"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	// 设置公网IP
	info.PublicIP = result.IP

	return nil
}

// getProxyStatus 获取网络代理状态
func getProxyStatus(info *model.NetworkInfo) error {
	// 使用networksetup命令获取网络代理状态
	output, err := runCommand("networksetup", "-getwebproxy", "Wi-Fi")
	if err != nil {
		return err
	}

	// 解析代理状态
	enabledRegex := regexp.MustCompile(`Enabled: (.+)`)
	matches := enabledRegex.FindStringSubmatch(output)

	if len(matches) > 1 {
		if matches[1] == "Yes" {
			// 如果代理已启用，获取代理服务器和端口
			serverRegex := regexp.MustCompile(`Server: (.+)`)
			portRegex := regexp.MustCompile(`Port: (.+)`)

			serverMatches := serverRegex.FindStringSubmatch(output)
			portMatches := portRegex.FindStringSubmatch(output)

			if len(serverMatches) > 1 && len(portMatches) > 1 {
				portNum, _ := strconv.Atoi(portMatches[1])
				info.ProxyInfo = model.ProxyInfo{
					Enabled: true,
					Server:  serverMatches[1],
					Port:    portNum,
				}
			}
		} else {
			info.ProxyInfo = model.ProxyInfo{
				Enabled: false,
			}
		}
	}

	return nil
}
