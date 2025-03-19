package darwin

import (
	"bufio"
	"encoding/json"
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

		// 解析RSSI (信号强度)
		if strings.Contains(line, "agrCtlRSSI: ") {
			parts := strings.Split(line, "agrCtlRSSI: ")
			if len(parts) > 1 {
				rssi, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.RSSI = rssi
					wifiInfo.SignalStrength = rssi
				}
			}
		}

		// 解析噪声
		if strings.Contains(line, "agrCtlNoise: ") {
			parts := strings.Split(line, "agrCtlNoise: ")
			if len(parts) > 1 {
				noise, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.Noise = noise
				}
			}
		}

		// 解析频道
		if strings.Contains(line, "channel: ") {
			parts := strings.Split(line, "channel: ")
			if len(parts) > 1 {
				channelInfo := strings.TrimSpace(parts[1])
				// 分离频道号和其他信息 (例如: "36,1" 或 "36,1 (5 GHz)")
				channelParts := strings.Split(channelInfo, ",")
				if len(channelParts) > 0 {
					channel, err := strconv.Atoi(strings.TrimSpace(channelParts[0]))
					if err == nil {
						wifiInfo.Channel = channel
					}
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

		// 解析PHY模式
		if strings.Contains(line, "MCS: ") {
			parts := strings.Split(line, "MCS: ")
			if len(parts) > 1 {
				mcsIndex, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.MSCIndex = mcsIndex
				}
			}
		}

		// 解析NSS (空间流数量)
		if strings.Contains(line, "NSS: ") {
			parts := strings.Split(line, "NSS: ")
			if len(parts) > 1 {
				nss, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					wifiInfo.NSS = nss
				}
			}
		}

		// 解析国家/地区代码
		if strings.Contains(line, "country: ") {
			parts := strings.Split(line, "country: ")
			if len(parts) > 1 {
				wifiInfo.CountryCode = strings.TrimSpace(parts[1])
			}
		}

		// 解析PHY模式
		if strings.Contains(line, "op mode: ") {
			parts := strings.Split(line, "op mode: ")
			if len(parts) > 1 {
				phyMode := strings.TrimSpace(parts[1])
				// 将操作模式转换为PHY模式
				switch {
				case strings.Contains(phyMode, "802.11ax"):
					wifiInfo.PHYMode = "802.11ax"
				case strings.Contains(phyMode, "802.11ac"):
					wifiInfo.PHYMode = "802.11ac"
				case strings.Contains(phyMode, "802.11n"):
					wifiInfo.PHYMode = "802.11n"
				case strings.Contains(phyMode, "802.11g"):
					wifiInfo.PHYMode = "802.11g"
				case strings.Contains(phyMode, "802.11b"):
					wifiInfo.PHYMode = "802.11b"
				case strings.Contains(phyMode, "802.11a"):
					wifiInfo.PHYMode = "802.11a"
				default:
					wifiInfo.PHYMode = phyMode
				}
			}
		}
	}

	// 获取支持的PHY模式
	supportedPHYOutput, err := runCommand("system_profiler", "SPAirPortDataType")
	if err == nil {
		phyModeRegex := regexp.MustCompile(`Supported PHY Modes:\s*(.+)`)
		matches := phyModeRegex.FindStringSubmatch(supportedPHYOutput)
		if len(matches) > 1 {
			wifiInfo.SupportedPHY = strings.TrimSpace(matches[1])
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
				iface.MACAddress = macMatches[1] // 兼容两种字段名
				interfaces[currentIface] = iface
			}

			// 解析IPv4地址
			ipv4Regex := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
			ipv4Matches := ipv4Regex.FindStringSubmatch(line)
			if len(ipv4Matches) > 1 {
				iface := interfaces[currentIface]
				iface.IP = ipv4Matches[1]

				// 添加到IP地址列表
				if iface.IPAddresses == nil {
					iface.IPAddresses = []string{}
				}
				iface.IPAddresses = append(iface.IPAddresses, ipv4Matches[1])

				interfaces[currentIface] = iface
			}

			// 解析IPv6地址
			ipv6Regex := regexp.MustCompile(`inet6\s+([0-9a-f:]+)`)
			ipv6Matches := ipv6Regex.FindStringSubmatch(line)
			if len(ipv6Matches) > 1 {
				iface := interfaces[currentIface]

				// 添加到IP地址列表
				if iface.IPAddresses == nil {
					iface.IPAddresses = []string{}
				}
				iface.IPAddresses = append(iface.IPAddresses, ipv6Matches[1])

				interfaces[currentIface] = iface
			}

			// 解析状态
			statusRegex := regexp.MustCompile(`status:\s+(.+)`)
			statusMatches := statusRegex.FindStringSubmatch(line)
			if len(statusMatches) > 1 {
				iface := interfaces[currentIface]
				status := statusMatches[1]
				iface.IsUp = status == "active"
				iface.Status = status
				interfaces[currentIface] = iface
			}
		}
	}

	// 将解析的接口信息添加到网络信息中
	var netInterfaces []model.NetInterfaceInfo
	for _, iface := range interfaces {
		netInterfaces = append(netInterfaces, iface)
	}
	info.Interfaces = netInterfaces

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
	// 使用ifconfig命令获取网络接口信息
	output, err := runCommand("ifconfig", "-a")
	if err != nil {
		return err
	}

	// 创建接口信息映射
	interfaces := make(map[string]model.NetInterfaceInfo)
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
				iface.MACAddress = macMatches[1] // 兼容两种字段名
				interfaces[currentIface] = iface
			}

			// 解析IPv4地址
			ipv4Regex := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
			ipv4Matches := ipv4Regex.FindStringSubmatch(line)
			if len(ipv4Matches) > 1 {
				iface := interfaces[currentIface]
				iface.IP = ipv4Matches[1]

				// 添加到IP地址列表
				if iface.IPAddresses == nil {
					iface.IPAddresses = []string{}
				}
				iface.IPAddresses = append(iface.IPAddresses, ipv4Matches[1])

				interfaces[currentIface] = iface
			}

			// 解析IPv6地址
			ipv6Regex := regexp.MustCompile(`inet6\s+([0-9a-f:]+)`)
			ipv6Matches := ipv6Regex.FindStringSubmatch(line)
			if len(ipv6Matches) > 1 {
				iface := interfaces[currentIface]

				// 添加到IP地址列表
				if iface.IPAddresses == nil {
					iface.IPAddresses = []string{}
				}
				iface.IPAddresses = append(iface.IPAddresses, ipv6Matches[1])

				interfaces[currentIface] = iface
			}

			// 解析状态
			statusRegex := regexp.MustCompile(`status:\s+(.+)`)
			statusMatches := statusRegex.FindStringSubmatch(line)
			if len(statusMatches) > 1 {
				iface := interfaces[currentIface]
				status := statusMatches[1]
				iface.IsUp = status == "active"
				iface.Status = status
				interfaces[currentIface] = iface
			}
		}
	}

	// 将解析的接口信息添加到网络信息中
	var netInterfaces []model.NetInterfaceInfo
	for _, iface := range interfaces {
		netInterfaces = append(netInterfaces, iface)
	}
	info.Interfaces = netInterfaces

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

	// 创建流量信息映射
	trafficByIface := make(map[string]model.NetworkTrafficInfo)

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

		// 解析接收和发送的字节数
		inBytes, err1 := strconv.ParseUint(fields[6], 10, 64)
		outBytes, err2 := strconv.ParseUint(fields[9], 10, 64)

		if err1 == nil && err2 == nil {
			// 更新接口的流量信息
			trafficInfo := model.NetworkTrafficInfo{
				Interface:     ifaceName,
				BytesReceived: inBytes,
				BytesSent:     outBytes,
			}
			trafficByIface[ifaceName] = trafficInfo
		}
	}

	// 更新接口的流量信息
	for i, iface := range info.Interfaces {
		if traffic, ok := trafficByIface[iface.Name]; ok {
			info.Interfaces[i].InBytes = traffic.BytesReceived
			info.Interfaces[i].OutBytes = traffic.BytesSent
		}
	}

	// 获取进程网络流量
	err = getProcessNetworkTraffic(info)
	if err != nil {
		log.Printf("Error getting process network traffic: %v", err)
	}

	return nil
}

// getProcessNetworkTraffic 获取进程网络流量
func getProcessNetworkTraffic(info *model.NetworkInfo) error {
	// 使用lsof命令获取进程网络连接信息
	output, err := runCommand("lsof", "-i", "-n", "-P")
	if err != nil {
		return err
	}

	// 创建进程流量映射
	processTraffic := make(map[string]*model.ProcessTrafficInfo)

	// 解析lsof输出
	scanner := bufio.NewScanner(strings.NewReader(output))
	
	// 跳过标题行
	scanner.Scan()
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		
		// 确保有足够的字段
		if len(fields) < 9 {
			continue
		}
		
		// 获取进程名称和PID
		processName := fields[0]
		pidStr := fields[1]
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		
		// 获取连接类型和地址
		networkInfo := fields[8]
		
		// 解析连接信息
		var connInfo model.ConnectionInfo
		
		// 检查是否是TCP或UDP连接
		if strings.Contains(networkInfo, "->") {
			// TCP连接
			connInfo.Protocol = "TCP"
			parts := strings.Split(networkInfo, "->")
			if len(parts) == 2 {
				// 解析本地地址和端口
				localParts := strings.Split(parts[0], ":")
				if len(localParts) == 2 {
					connInfo.LocalAddr = localParts[0]
					connInfo.LocalPort, _ = strconv.Atoi(localParts[1])
				}
				
				// 解析远程地址和端口
				remoteParts := strings.Split(parts[1], ":")
				if len(remoteParts) == 2 {
					connInfo.RemoteAddr = remoteParts[0]
					connInfo.RemotePort, _ = strconv.Atoi(remoteParts[1])
				}
			}
			
			// 设置连接状态
			if len(fields) > 9 {
				connInfo.State = fields[9]
			}
		} else if strings.Contains(networkInfo, "UDP") {
			// UDP连接
			connInfo.Protocol = "UDP"
			parts := strings.Split(networkInfo, ":")
			if len(parts) == 2 {
				connInfo.LocalAddr = parts[0]
				connInfo.LocalPort, _ = strconv.Atoi(parts[1])
			}
			connInfo.State = "ESTABLISHED"
		} else {
			continue
		}
		
		// 获取或创建进程流量信息
		var process *model.ProcessTrafficInfo
		if p, ok := processTraffic[pidStr]; ok {
			process = p
		} else {
			process = &model.ProcessTrafficInfo{
				PID:           pid,
				Name:          processName,
				ProcessName:   processName,
				BytesIn:       0,
				BytesOut:      0,
				Connections:   []model.ConnectionInfo{},
				ConnectionCount: 0,
			}
			processTraffic[pidStr] = process
		}
		
		// 添加连接信息
		process.Connections = append(process.Connections, connInfo)
		process.ConnectionCount++
	}
	
	// 使用nettop命令获取进程流量信息（如果可用）
	nettopOutput, err := runCommand("nettop", "-P", "-L", "1", "-n", "-J", "bytes_in,bytes_out")
	if err == nil {
		scanner = bufio.NewScanner(strings.NewReader(nettopOutput))
		
		// 跳过标题行
		for i := 0; i < 2 && scanner.Scan(); i++ {}
		
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			
			// 确保有足够的字段
			if len(fields) < 5 {
				continue
			}
			
			// 获取进程名称、PID和流量信息
			processName := fields[0]
			pidStr := fields[1]
			
			// 尝试解析流量数据
			var bytesIn, bytesOut uint64
			if len(fields) > 3 {
				bytesIn, _ = strconv.ParseUint(fields[3], 10, 64)
			}
			if len(fields) > 4 {
				bytesOut, _ = strconv.ParseUint(fields[4], 10, 64)
			}
			
			// 更新进程流量信息
			if process, ok := processTraffic[pidStr]; ok {
				process.BytesIn += bytesIn
				process.BytesOut += bytesOut
			} else if bytesIn > 0 || bytesOut > 0 {
				pid, _ := strconv.Atoi(pidStr)
				process := &model.ProcessTrafficInfo{
					PID:         pid,
					Name:        processName,
					ProcessName: processName,
					BytesIn:     bytesIn,
					BytesOut:    bytesOut,
					Connections: []model.ConnectionInfo{},
				}
				processTraffic[pidStr] = process
			}
		}
	}
	
	// 将进程流量信息添加到网络信息中
	processTrafficList := []model.ProcessTrafficInfo{}
	for _, process := range processTraffic {
		processTrafficList = append(processTrafficList, *process)
	}
	
	// 更新流量信息
	info.Traffic.ProcessTraffic = processTrafficList
	
	return nil
}

// getNetworkLatency 获取网络延迟信息
func getNetworkLatency(info *model.NetworkInfo) error {
	// 初始化延迟信息
	latencyInfo := model.LatencyInfo{
		Targets:     []model.TargetLatencyInfo{},
		NetworkHops: []model.NetworkHopInfo{},
	}
	
	// 定义要ping的目标
	targets := []struct {
		Name string
		Host string
	}{
		{"Google DNS", "8.8.8.8"},
		{"Cloudflare DNS", "1.1.1.1"},
		{"Baidu", "www.baidu.com"},
	}
	
	for _, target := range targets {
		// 使用ping命令获取网络延迟信息
		output, err := runCommand("ping", "-c", "5", "-q", target.Host)
		if err != nil {
			log.Printf("Error pinging %s: %v", target.Host, err)
			continue
		}
		
		// 解析ping结果
		latencyRegex := regexp.MustCompile(`min/avg/max/stddev = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+)`)
		matches := latencyRegex.FindStringSubmatch(output)
		
		if len(matches) > 4 {
			// 解析延迟数据
			min, _ := strconv.ParseFloat(matches[1], 64)
			avg, _ := strconv.ParseFloat(matches[2], 64)
			max, _ := strconv.ParseFloat(matches[3], 64)
			stddev, _ := strconv.ParseFloat(matches[4], 64)
			
			// 解析丢包率
			packetLossRegex := regexp.MustCompile(`(\d+)% packet loss`)
			plMatches := packetLossRegex.FindStringSubmatch(output)
			var packetLoss float64
			if len(plMatches) > 1 {
				packetLoss, _ = strconv.ParseFloat(plMatches[1], 64)
			}
			
			// 创建目标延迟信息
			targetLatency := model.TargetLatencyInfo{
				TargetName:  target.Name,
				TargetHost:  target.Host,
				MinLatency:  min,
				AvgLatency:  avg,
				MaxLatency:  max,
				StdDev:      stddev,
				PacketLoss:  packetLoss,
				Jitter:      stddev, // 使用标准差作为抖动的估计值
			}
			
			// 添加到延迟信息中
			latencyInfo.Targets = append(latencyInfo.Targets, targetLatency)
		}
	}
	
	// 使用mtr命令获取更详细的网络路径信息（如果可用）
	mtrOutput, err := runCommand("mtr", "-r", "-c", "5", "8.8.8.8")
	if err == nil {
		// 解析mtr输出
		scanner := bufio.NewScanner(strings.NewReader(mtrOutput))
		
		// 跳过标题行
		for i := 0; i < 2 && scanner.Scan(); i++ {}
		
		var hops []model.NetworkHopInfo
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			
			// 确保有足够的字段
			if len(fields) < 10 {
				continue
			}
			
			// 解析跳数和主机名
			hopNum, _ := strconv.Atoi(fields[0])
			hostname := fields[1]
			
			// 解析丢包率和延迟
			lossStr := fields[2]
			loss, _ := strconv.ParseFloat(strings.TrimSuffix(lossStr, "%"), 64)
			
			snt, _ := strconv.Atoi(fields[3])
			last, _ := strconv.ParseFloat(fields[4], 64)
			avg, _ := strconv.ParseFloat(fields[5], 64)
			best, _ := strconv.ParseFloat(fields[6], 64)
			worst, _ := strconv.ParseFloat(fields[7], 64)
			stddev, _ := strconv.ParseFloat(fields[8], 64)
			
			// 创建跳点信息
			hop := model.NetworkHopInfo{
				HopNum:       hopNum,
				Host:         hostname,
				Loss:         loss,
				SentPackets:  snt,
				LastLatency:  last,
				AvgLatency:   avg,
				BestLatency:  best,
				WorstLatency: worst,
				StdDev:       stddev,
			}
			
			hops = append(hops, hop)
		}
		
		latencyInfo.NetworkHops = hops
	}
	
	// 设置网络延迟信息
	info.Latency = latencyInfo
	
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

	// 如果使用了Cisco AnyConnect VPN，尝试获取其状态
	anyconnectOutput, err := runCommand("/opt/cisco/anyconnect/bin/vpn", "state")
	if err == nil && !strings.Contains(anyconnectOutput, "not found") {
		if strings.Contains(anyconnectOutput, "state: Connected") {
			vpnInfo.IsConnected = true
			vpnInfo.Provider = "Cisco AnyConnect"
			
			// 提取连接的服务器
			serverRegex := regexp.MustCompile(`>> server\s*:\s*(.+)`)
			matches := serverRegex.FindStringSubmatch(anyconnectOutput)
			if len(matches) > 1 {
				vpnInfo.Server = matches[1]
				vpnInfo.Nodes = append(vpnInfo.Nodes, matches[1])
			}
		}
	}

	// 如果使用了OpenVPN，尝试获取其状态
	openvpnOutput, err := runCommand("ps", "-ef")
	if err == nil {
		if strings.Contains(openvpnOutput, "openvpn") {
			// 提取OpenVPN配置文件路径
			openvpnRegex := regexp.MustCompile(`openvpn\s+--config\s+([^\s]+)`)
			matches := openvpnRegex.FindStringSubmatch(openvpnOutput)
			if len(matches) > 1 {
				vpnInfo.IsConnected = true
				vpnInfo.Provider = "OpenVPN"
				vpnInfo.ConfigFile = matches[1]
				
				// 尝试从配置文件中提取服务器信息
				if _, err := os.Stat(matches[1]); err == nil {
					configContent, err := os.ReadFile(matches[1])
					if err == nil {
						serverRegex := regexp.MustCompile(`remote\s+([^\s]+)`)
						serverMatches := serverRegex.FindAllSubmatch(configContent, -1)
						for _, match := range serverMatches {
							if len(match) > 1 {
								server := string(match[1])
								vpnInfo.Nodes = append(vpnInfo.Nodes, server)
								if vpnInfo.Server == "" {
									vpnInfo.Server = server
								}
							}
						}
					}
				}
			}
		}
	}

	// 设置VPN信息
	info.VPN = vpnInfo

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
