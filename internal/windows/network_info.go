//go:build windows
// +build windows

package windows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AsterZephyr/SysSpector/pkg/model"
	"github.com/shirou/gopsutil/v3/net"
)

// 定义WMI查询结构体
type win32NetworkAdapter struct {
	Name                 string
	NetConnectionID      string
	MACAddress           string
	Speed                uint64
	AdapterType          string
	PhysicalAdapter      bool
	NetEnabled           bool
	ProductName          string
	ServiceName          string
	DHCPEnabled          bool
	IPAddress            []string
	IPSubnet             []string
	DefaultIPGateway     []string
	DNSServerSearchOrder []string
}

type win32NetworkAdapterConfiguration struct {
	Description          string
	DHCPEnabled          bool
	IPAddress            []string
	IPSubnet             []string
	DefaultIPGateway     []string
	DNSServerSearchOrder []string
	MACAddress           string
}

// GetNetworkInfo 获取Windows系统的网络信息
func GetNetworkInfo() (model.NetworkInfo, error) {
	var info model.NetworkInfo
	var err error

	// 获取网络适配器信息
	var adapters []win32NetworkAdapter
	err = safeWMIQuery("SELECT Name, NetConnectionID, MACAddress, Speed, AdapterType, PhysicalAdapter, NetEnabled, ProductName, ServiceName, DHCPEnabled, IPAddress, IPSubnet, DefaultIPGateway, DNSServerSearchOrder FROM Win32_NetworkAdapter WHERE PhysicalAdapter=True", &adapters)
	
	if err != nil || len(adapters) == 0 {
		log.Printf("Error getting network adapters or no adapters found: %v", err)
	} else {
		// 找到活跃的网络适配器
		for _, adapter := range adapters {
			if adapter.NetEnabled && adapter.PhysicalAdapter {
				// 获取IP地址
				if len(adapter.IPAddress) > 0 {
					info.IP = adapter.IPAddress[0]
				}
				
				// 获取MAC地址
				info.MacAddress = adapter.MACAddress
				
				// 获取子网掩码
				if len(adapter.IPSubnet) > 0 {
					info.SubnetMask = adapter.IPSubnet[0]
				}
				
				// 获取网关
				if len(adapter.DefaultIPGateway) > 0 {
					info.Gateway = adapter.DefaultIPGateway[0]
				}
				
				// 获取DNS服务器
				info.DNSServers = adapter.DNSServerSearchOrder
				
				// 获取IP获取方式
				if adapter.DHCPEnabled {
					info.IPAcquisitionMode = "DHCP"
				} else {
					info.IPAcquisitionMode = "静态IP"
				}
				
				// 设置接口名称
				info.InterfaceName = adapter.NetConnectionID
				
				// 设置WiFi连接状态
				if strings.Contains(adapter.Name, "Wireless") || strings.Contains(adapter.Name, "WiFi") || strings.Contains(adapter.Name, "Wi-Fi") {
					info.WiFi.IsConnected = adapter.NetEnabled
				}
				
				break
			}
		}
	}
	
	// 获取公网IP
	info.PublicIP = getPublicIP()
	info.PublicIPSource = "通过外部API获取"
	info.PublicIPDetails = "使用https://api.ipify.org服务"
	
	// 获取网络延迟、抖动和丢包信息
	getNetworkLatency(&info)
	
	// 获取VPN状态
	getVPNInfo(&info)
	
	// 获取网络代理状态
	getProxyStatus(&info)
	
	// 获取路由表
	info.RouteTable = getRouteTable()
	
	// 获取Hosts文件
	hostEntries := getHostsFile()
	if len(hostEntries) > 0 {
		info.DNS.HostEntries = hostEntries
	}
	
	// 获取国家/地区代码
	info.CountryCode = getCountryCode()
	
	// 获取WiFi信息
	wifiInfo, err := getWiFiInfo()
	if err == nil {
		info.WiFi = wifiInfo
	}
	
	// 获取网络流量
	info.NetworkTraffic = getNetworkTraffic()
	
	return info, nil
}

// getNetworkLatency 获取网络延迟、抖动和丢包信息
func getNetworkLatency(info *model.NetworkInfo) {
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
		cmd := exec.Command("ping", "-n", "5", target.host)
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}

		// 解析ping输出
		outputStr := string(output)
		targetInfo := model.TargetLatencyInfo{
			TargetName: target.name,
			TargetHost: target.host,
		}

		// 提取延迟信息
		// 示例输出: 最短 = 4ms，最长 = 6ms，平均 = 5ms
		minRegex := regexp.MustCompile(`最短 = (\d+)ms`)
		maxRegex := regexp.MustCompile(`最长 = (\d+)ms`)
		avgRegex := regexp.MustCompile(`平均 = (\d+)ms`)
		
		minMatches := minRegex.FindStringSubmatch(outputStr)
		maxMatches := maxRegex.FindStringSubmatch(outputStr)
		avgMatches := avgRegex.FindStringSubmatch(outputStr)
		
		if len(minMatches) >= 2 && len(maxMatches) >= 2 && len(avgMatches) >= 2 {
			min, _ := strconv.ParseFloat(minMatches[1], 64)
			max, _ := strconv.ParseFloat(maxMatches[1], 64)
			avg, _ := strconv.ParseFloat(avgMatches[1], 64)
			
			targetInfo.MinLatency = min
			targetInfo.MaxLatency = max
			targetInfo.AvgLatency = avg
			
			// 计算抖动（使用最大值和最小值的差值作为抖动的估计值）
			targetInfo.Jitter = max - min
			
			// 累加总延迟
			totalLatency += targetInfo.AvgLatency
			validTargetCount++
		}

		// 提取丢包率
		// 示例输出: 已发送 = 5，已接收 = 5，丢失 = 0 (0% 丢失)
		lossRegex := regexp.MustCompile(`丢失 = \d+ \((\d+)% 丢失\)`)
		lossMatches := lossRegex.FindStringSubmatch(outputStr)
		if len(lossMatches) >= 2 {
			targetInfo.PacketLoss, _ = strconv.ParseFloat(lossMatches[1], 64)
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
}

// getVPNInfo 获取VPN信息
func getVPNInfo(info *model.NetworkInfo) {
	// 初始化VPN信息
	vpnInfo := model.VPNInfo{
		IsConnected: false,
		Services:    []string{},
		Nodes:       []string{},
	}

	// 检查VPN适配器
	var adapters []win32NetworkAdapter
	err := safeWMIQuery("SELECT Name, NetConnectionID, MACAddress, Speed, AdapterType, PhysicalAdapter, NetEnabled, ProductName, ServiceName FROM Win32_NetworkAdapter WHERE (Name LIKE '%VPN%' OR Name LIKE '%Virtual%' OR Name LIKE '%Tunnel%') AND NetEnabled=True", &adapters)
	
	if err == nil && len(adapters) > 0 {
		for _, adapter := range adapters {
			if adapter.NetEnabled {
				vpnInfo.IsConnected = true
				vpnInfo.Interfaces = append(vpnInfo.Interfaces, adapter.NetConnectionID)
				vpnInfo.NodeName = adapter.NetConnectionID
				break
			}
		}
	}

	// 检查常见VPN客户端进程
	vpnClients := map[string]string{
		"Cisco AnyConnect": "vpnui.exe",
		"OpenVPN":          "openvpn.exe",
		"WireGuard":        "wireguard.exe",
		"NordVPN":          "nordvpn.exe",
		"ExpressVPN":       "expressvpn.exe",
		"Tunnelblick":      "tunnelblick.exe",
		"Viscosity":        "viscosity.exe",
		"Pritunl":          "pritunl.exe",
		"Pulse Secure":     "pulsetray.exe",
		"FortiClient":      "forticlient.exe",
		"Kit":              "kit.exe",
		"ZeroPass":         "zeropass.exe",
	}

	for clientName, processName := range vpnClients {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), processName) {
			vpnInfo.IsConnected = true
			vpnInfo.ClientName = clientName
			vpnInfo.NodeName = clientName + " VPN"
			
			// 如果是Kit应用，获取更多详细信息
			if clientName == "Kit" || clientName == "ZeroPass" {
				vpnInfo.NodeName = "Kit VPN (ZeroPass)"
			}
			
			break
		}
	}

	// 设置VPN信息
	info.VPN = vpnInfo
}

// getProxyStatus 获取网络代理状态
func getProxyStatus(info *model.NetworkInfo) {
	// 初始化代理信息
	info.ProxyInfo = model.ProxyInfo{
		Enabled: false,
	}
	
	// 检查系统代理设置
	cmd := exec.Command("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings", "/v", "ProxyEnable")
	output, err := cmd.CombinedOutput()
	
	if err == nil && strings.Contains(string(output), "0x1") {
		// 代理已启用
		info.ProxyInfo.Enabled = true
		info.ProxyStatus = true
		
		// 获取代理服务器地址
		cmd = exec.Command("reg", "query", "HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings", "/v", "ProxyServer")
		output, err = cmd.CombinedOutput()
		
		if err == nil {
			// 提取代理服务器地址
			proxyRegex := regexp.MustCompile(`ProxyServer\s+REG_SZ\s+(.+)`)
			matches := proxyRegex.FindStringSubmatch(string(output))
			
			if len(matches) > 1 {
				proxyServer := strings.TrimSpace(matches[1])
				
				// 检查是否包含端口
				if strings.Contains(proxyServer, ":") {
					parts := strings.Split(proxyServer, ":")
					if len(parts) == 2 {
						info.ProxyInfo.Server = parts[0]
						port, _ := strconv.Atoi(parts[1])
						info.ProxyInfo.Port = port
					}
				} else {
					info.ProxyInfo.Server = proxyServer
					info.ProxyInfo.Port = 80 // 默认端口
				}
			}
		}
	}
	
	// 检查是否有代理软件正在运行
	proxyApps := []string{
		"Proxifier.exe", "Charles.exe", "Fiddler.exe", "mitmproxy.exe", "Burp.exe", "tinyproxy.exe",
		"Surge.exe", "ClashX.exe", "v2ray.exe", "Shadowsocks.exe", "Lantern.exe", "Outline.exe",
	}
	
	for _, app := range proxyApps {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", app))
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), app) {
			info.ProxyInfo.Enabled = true
			info.ProxyInfo.ProxyAppRunning = true
			info.ProxyInfo.ProxyAppName = strings.TrimSuffix(app, ".exe")
			info.ProxyStatus = true
			break
		}
	}
}

// getPublicIP 获取公网IP
func getPublicIP() string {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	// 尝试多个API获取公网IP
	apis := []string{
		"https://api.ipify.org",
		"https://ipinfo.io/ip",
		"https://api.ip.sb/ip",
	}
	
	for _, api := range apis {
		resp, err := client.Get(api)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			
			ip := strings.TrimSpace(string(body))
			// 简单验证IP格式
			if strings.Count(ip, ".") == 3 {
				return ip
			}
		}
	}
	
	return ""
}

// getRouteTable 获取路由表
func getRouteTable() []model.RouteEntry {
	var routes []model.RouteEntry
	
	// 使用route print命令获取路由表
	cmd := exec.Command("route", "print")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error getting route table: %v", err)
		return routes
	}
	
	// 解析输出
	lines := strings.Split(string(output), "\n")
	inIPv4Section := false
	headerFound := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 识别IPv4路由表部分
		if strings.Contains(line, "IPv4 Route Table") {
			inIPv4Section = true
			continue
		}
		
		// 识别IPv6路由表部分（结束IPv4部分）
		if inIPv4Section && strings.Contains(line, "IPv6 Route Table") {
			break
		}
		
		// 跳过空行
		if inIPv4Section && line == "" {
			continue
		}
		
		// 识别表头行
		if inIPv4Section && strings.Contains(line, "Network Destination") {
			headerFound = true
			continue
		}
		
		// 解析路由条目
		if inIPv4Section && headerFound && len(line) > 0 {
			fields := regexp.MustCompile(`\s+`).Split(line, -1)
			if len(fields) >= 5 {
				routes = append(routes, model.RouteEntry{
					Destination: fields[0],
					Gateway:     fields[1],
					Flags:       fields[3], // 使用Metric作为Flags
					Interface:   fields[4],
					Netmask:     fields[2], // 使用Genmask作为Netmask
				})
			}
		}
	}
	
	return routes
}

// getHostsFile 获取Hosts文件内容
func getHostsFile() []model.HostEntry {
	var hosts []model.HostEntry
	
	// 读取hosts文件
	hostsPath := os.Getenv("SystemRoot") + "\\System32\\drivers\\etc\\hosts"
	content, err := ioutil.ReadFile(hostsPath)
	if err != nil {
		log.Printf("Error reading hosts file: %v", err)
		return hosts
	}
	
	// 解析hosts文件
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// 解析IP和主机名
		fields := regexp.MustCompile(`\s+`).Split(line, -1)
		if len(fields) >= 2 {
			ip := fields[0]
			for _, hostname := range fields[1:] {
				if hostname != "" && !strings.HasPrefix(hostname, "#") {
					hosts = append(hosts, model.HostEntry{
						IP:       ip,
						Hostname: hostname,
					})
				}
			}
		}
	}
	
	return hosts
}

// getCountryCode 获取国家/地区代码
func getCountryCode() string {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get("http://ip-api.com/json/")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	
	var result struct {
		CountryCode string `json:"countryCode"`
	}
	
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return ""
	}
	
	return result.CountryCode
}

// getWiFiInfo 获取WiFi信息
func getWiFiInfo() (model.WiFiInfo, error) {
	var wifiInfo model.WiFiInfo
	
	// 使用netsh命令获取WiFi信息
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	output, err := cmd.Output()
	if err != nil {
		return wifiInfo, fmt.Errorf("error getting WiFi info: %v", err)
	}
	
	// 解析输出
	outputStr := string(output)
	
	// 提取SSID
	ssidRegex := regexp.MustCompile(`SSID\s+:\s+(.+)`)
	ssidMatches := ssidRegex.FindStringSubmatch(outputStr)
	if len(ssidMatches) > 1 {
		wifiInfo.SSID = strings.TrimSpace(ssidMatches[1])
	}
	
	// 提取BSSID
	bssidRegex := regexp.MustCompile(`BSSID\s+:\s+(.+)`)
	bssidMatches := bssidRegex.FindStringSubmatch(outputStr)
	if len(bssidMatches) > 1 {
		wifiInfo.BSSID = strings.TrimSpace(bssidMatches[1])
	}
	
	// 提取信号强度
	signalRegex := regexp.MustCompile(`Signal\s+:\s+(\d+)%`)
	signalMatches := signalRegex.FindStringSubmatch(outputStr)
	if len(signalMatches) > 1 {
		signalStr := strings.TrimSpace(signalMatches[1])
		signal, _ := strconv.Atoi(signalStr)
		// 将百分比转换为dBm（近似值）
		// 信号强度通常在-100dBm到-30dBm之间
		// 100%约等于-30dBm，0%约等于-100dBm
		rssi := -30 - (100-signal)*70/100
		wifiInfo.RSSI = rssi
		wifiInfo.SignalStrength = rssi
	}
	
	// 提取频道
	channelRegex := regexp.MustCompile(`Channel\s+:\s+(\d+)`)
	channelMatches := channelRegex.FindStringSubmatch(outputStr)
	if len(channelMatches) > 1 {
		channel := strings.TrimSpace(channelMatches[1])
		channelNum, _ := strconv.Atoi(channel)
		wifiInfo.Channel = channelNum
		
		// 确定频段（2.4GHz或5GHz）
		if channelNum > 14 {
			wifiInfo.Frequency = 5.0
		} else {
			wifiInfo.Frequency = 2.4
		}
	}
	
	// 提取PHY模式
	radioTypeRegex := regexp.MustCompile(`Radio type\s+:\s+(.+)`)
	radioTypeMatches := radioTypeRegex.FindStringSubmatch(outputStr)
	if len(radioTypeMatches) > 1 {
		radioType := strings.TrimSpace(radioTypeMatches[1])
		
		// 将Windows的无线电类型映射到PHY模式
		phyModeMap := map[string]string{
			"802.11n": "802.11n",
			"802.11ac": "802.11ac",
			"802.11ax": "802.11ax",
			"802.11a": "802.11a",
			"802.11g": "802.11g",
			"802.11b": "802.11b",
		}
		
		for key, value := range phyModeMap {
			if strings.Contains(radioType, key) {
				wifiInfo.PHYMode = value
				break
			}
		}
		
		// 如果没有匹配到，使用原始值
		if wifiInfo.PHYMode == "" {
			wifiInfo.PHYMode = radioType
		}
	}
	
	// 获取支持的PHY模式
	cmd = exec.Command("netsh", "wlan", "show", "drivers")
	output, err = cmd.Output()
	if err == nil {
		outputStr = string(output)
		
		// 提取支持的无线模式
		supportedRegex := regexp.MustCompile(`Supported\s+802.11\s+protocols\s+:\s+(.+)`)
		supportedMatches := supportedRegex.FindStringSubmatch(outputStr)
		if len(supportedMatches) > 1 {
			supported := strings.TrimSpace(supportedMatches[1])
			
			// 格式化为与macOS版本相似的格式
			modes := []string{}
			if strings.Contains(supported, "a") {
				modes = append(modes, "a")
			}
			if strings.Contains(supported, "b") {
				modes = append(modes, "b")
			}
			if strings.Contains(supported, "g") {
				modes = append(modes, "g")
			}
			if strings.Contains(supported, "n") {
				modes = append(modes, "n")
			}
			if strings.Contains(supported, "ac") {
				modes = append(modes, "ac")
			}
			if strings.Contains(supported, "ax") {
				modes = append(modes, "ax")
			}
			
			if len(modes) > 0 {
				wifiInfo.SupportedPHY = "802.11 " + strings.Join(modes, "/")
			} else {
				wifiInfo.SupportedPHY = supported
			}
		}
	}
	
	// 获取WiFi国家/地区代码
	cmd = exec.Command("netsh", "wlan", "show", "settings")
	output, err = cmd.Output()
	if err == nil {
		outputStr = string(output)
		
		// 提取国家/地区代码
		countryRegex := regexp.MustCompile(`Country or region\s+:\s+(.+)`)
		countryMatches := countryRegex.FindStringSubmatch(outputStr)
		if len(countryMatches) > 1 {
			country := strings.TrimSpace(countryMatches[1])
			
			// 提取国家/地区代码（通常是括号中的内容）
			codeRegex := regexp.MustCompile(`\((.+)\)`)
			codeMatches := codeRegex.FindStringSubmatch(country)
			if len(codeMatches) > 1 {
				wifiInfo.CountryCode = codeMatches[1]
			} else {
				wifiInfo.CountryCode = country
			}
		}
	}
	
	// 获取传输速率
	txRateRegex := regexp.MustCompile(`Transmit\s+rate\s+\(Mbps\)\s+:\s+(.+)`)
	txRateMatches := txRateRegex.FindStringSubmatch(outputStr)
	if len(txRateMatches) > 1 {
		txRate := strings.TrimSpace(txRateMatches[1])
		txRateNum, _ := strconv.Atoi(txRate)
		wifiInfo.TxRate = txRateNum
	}
	
	return wifiInfo, nil
}

// getNetworkTraffic 获取网络流量
func getNetworkTraffic() string {
	// 获取当前网络流量
	counters, err := net.IOCounters(true)
	if err != nil {
		return ""
	}
	
	// 记录第一次采样
	var activeInterface net.IOCountersStat
	found := false
	
	// 查找活跃的网络接口
	for _, counter := range counters {
		if counter.BytesSent > 0 || counter.BytesRecv > 0 {
			activeInterface = counter
			found = true
			break
		}
	}
	
	if !found {
		return "0 KB/s"
	}
	
	// 等待1秒进行第二次采样
	time.Sleep(1 * time.Second)
	
	// 获取第二次采样
	counters, err = net.IOCounters(true)
	if err != nil {
		return ""
	}
	
	// 计算流量差值
	for _, counter := range counters {
		if counter.Name == activeInterface.Name {
			sentDiff := float64(counter.BytesSent - activeInterface.BytesSent)
			recvDiff := float64(counter.BytesRecv - activeInterface.BytesRecv)
			
			// 计算总流量（KB/s）
			totalKBps := (sentDiff + recvDiff) / 1024
			
			return fmt.Sprintf("%.2f KB/s", totalKBps)
		}
	}
	
	return "0 KB/s"
}

// getVPNStatus 获取VPN状态
func getVPNStatus() string {
	// 使用netsh命令检查VPN连接
	cmd := exec.Command("netsh", "interface", "show", "interface")
	output, err := cmd.Output()
	if err != nil {
		return "未连接"
	}
	
	// 检查输出中是否包含VPN接口
	outputStr := string(output)
	if strings.Contains(outputStr, "VPN") || strings.Contains(outputStr, "PPP") {
		return "已连接"
	}
	
	return "未连接"
}
