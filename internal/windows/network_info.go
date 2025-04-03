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

	// 首先使用Win32_NetworkAdapter获取网络适配器基本信息
	var adapters []win32NetworkAdapter
	err = safeWMIQuery("SELECT Name, NetConnectionID, MACAddress, AdapterType, PhysicalAdapter, NetEnabled, ProductName, ServiceName FROM Win32_NetworkAdapter WHERE PhysicalAdapter = 'TRUE'", &adapters)
	
	// 然后使用Win32_NetworkAdapterConfiguration获取IP配置信息
	var adapterConfigs []win32NetworkAdapterConfiguration
	configErr := safeWMIQuery("SELECT Description, DHCPEnabled, IPAddress, IPSubnet, DefaultIPGateway, DNSServerSearchOrder, MACAddress FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled = 'TRUE'", &adapterConfigs)
	
	// 如果WMI查询失败，使用备用方法
	if (err != nil || len(adapters) == 0) && (configErr != nil || len(adapterConfigs) == 0) {
		log.Printf("Error getting network adapters or configurations: %v, %v", err, configErr)
		
		// 使用ipconfig命令获取详细网络信息
		cmd := exec.Command("ipconfig", "/all")
		output, err := cmd.Output()
		if err == nil {
			outputStr := string(output)

			// 获取IP地址
			ipRegex := regexp.MustCompile(`IPv4 地址.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "IPv4 地址") {
				// 英文系统
				ipRegex = regexp.MustCompile(`IPv4 Address.*?:\s*(.+)`)
			}
			ipMatches := ipRegex.FindStringSubmatch(outputStr)
			if len(ipMatches) > 1 {
				ip := strings.TrimSpace(ipMatches[1])
				// 移除可能的括号内容
				if idx := strings.Index(ip, "("); idx > 0 {
					ip = strings.TrimSpace(ip[:idx])
				}
				info.IP = ip
			}

			// 获取子网掩码
			maskRegex := regexp.MustCompile(`子网掩码.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "子网掩码") {
				// 英文系统
				maskRegex = regexp.MustCompile(`Subnet Mask.*?:\s*(.+)`)
			}
			maskMatches := maskRegex.FindStringSubmatch(outputStr)
			if len(maskMatches) > 1 {
				info.SubnetMask = strings.TrimSpace(maskMatches[1])
			}

			// 获取默认网关
			gatewayRegex := regexp.MustCompile(`默认网关.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "默认网关") {
				// 英文系统
				gatewayRegex = regexp.MustCompile(`Default Gateway.*?:\s*(.+)`)
			}
			gatewayMatches := gatewayRegex.FindStringSubmatch(outputStr)
			if len(gatewayMatches) > 1 {
				info.Gateway = strings.TrimSpace(gatewayMatches[1])
			}

			// 获取DNS服务器
			dnsRegex := regexp.MustCompile(`DNS 服务器.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "DNS 服务器") {
				// 英文系统
				dnsRegex = regexp.MustCompile(`DNS Servers.*?:\s*(.+)`)
			}
			dnsMatches := dnsRegex.FindAllStringSubmatch(outputStr, -1)
			for _, match := range dnsMatches {
				if len(match) > 1 {
					dns := strings.TrimSpace(match[1])
					if dns != "" && !strings.Contains(dns, "::") { // 排除IPv6地址
						info.DNSServers = append(info.DNSServers, dns)
					}
				}
			}

			// 获取IP获取方式
			dhcpRegex := regexp.MustCompile(`DHCP 已启用.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "DHCP 已启用") {
				// 英文系统
				dhcpRegex = regexp.MustCompile(`DHCP Enabled.*?:\s*(.+)`)
			}
			dhcpMatches := dhcpRegex.FindStringSubmatch(outputStr)
			if len(dhcpMatches) > 1 {
				dhcpEnabled := strings.TrimSpace(dhcpMatches[1])
				if strings.ToLower(dhcpEnabled) == "yes" || strings.ToLower(dhcpEnabled) == "是" {
					info.IPAcquisitionMode = "DHCP"
				} else {
					info.IPAcquisitionMode = "静态IP"
				}
			}

			// 获取物理地址(MAC)
			macRegex := regexp.MustCompile(`物理地址.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "物理地址") {
				// 英文系统
				macRegex = regexp.MustCompile(`Physical Address.*?:\s*(.+)`)
			}
			macMatches := macRegex.FindStringSubmatch(outputStr)
			if len(macMatches) > 1 {
				info.MacAddress = strings.TrimSpace(macMatches[1])
			}

			// 获取接口名称
			nameRegex := regexp.MustCompile(`适配器名称.*?:\s*(.+)`)
			if !strings.Contains(outputStr, "适配器名称") {
				// 英文系统
				nameRegex = regexp.MustCompile(`Adapter Name.*?:\s*(.+)`)
			}
			nameMatches := nameRegex.FindStringSubmatch(outputStr)
			if len(nameMatches) > 1 {
				info.InterfaceName = strings.TrimSpace(nameMatches[1])
			}

			// 检测是否连接了WiFi
			if strings.Contains(outputStr, "Wireless") || 
			   strings.Contains(outputStr, "Wi-Fi") || 
			   strings.Contains(outputStr, "无线") {
				info.WiFi.IsConnected = true
			}
		}
	} else {
		// WMI查询成功，处理查询结果
		
		// 处理网络适配器基本信息
		var activeAdapter *win32NetworkAdapter
		for i := range adapters {
			if adapters[i].NetEnabled && adapters[i].PhysicalAdapter {
				activeAdapter = &adapters[i]
				
				// 设置基本信息
				info.MacAddress = adapters[i].MACAddress
				info.InterfaceName = adapters[i].NetConnectionID
				
				// 设置WiFi连接状态
				if strings.Contains(adapters[i].Name, "Wireless") || 
				   strings.Contains(adapters[i].Name, "WiFi") || 
				   strings.Contains(adapters[i].Name, "Wi-Fi") {
					info.WiFi.IsConnected = adapters[i].NetEnabled
				}
				
				break
			}
		}
		
		// 处理网络适配器配置信息
		if configErr == nil && len(adapterConfigs) > 0 {
			// 如果找到了活跃的适配器，尝试匹配对应的配置
			if activeAdapter != nil {
				for _, config := range adapterConfigs {
					if config.MACAddress == activeAdapter.MACAddress {
						// 获取IP地址
						if len(config.IPAddress) > 0 {
							info.IP = config.IPAddress[0]
						}
						
						// 获取子网掩码
						if len(config.IPSubnet) > 0 {
							info.SubnetMask = config.IPSubnet[0]
						}
						
						// 获取网关
						if len(config.DefaultIPGateway) > 0 {
							info.Gateway = config.DefaultIPGateway[0]
						}
						
						// 获取DNS服务器
						info.DNSServers = config.DNSServerSearchOrder
						
						// 获取IP获取方式
						if config.DHCPEnabled {
							info.IPAcquisitionMode = "DHCP"
						} else {
							info.IPAcquisitionMode = "静态IP"
						}
						
						break
					}
				}
			} else {
				// 如果没有找到活跃的适配器，使用第一个配置
				if len(adapterConfigs) > 0 {
					// 获取IP地址
					if len(adapterConfigs[0].IPAddress) > 0 {
						info.IP = adapterConfigs[0].IPAddress[0]
					}
					
					// 获取子网掩码
					if len(adapterConfigs[0].IPSubnet) > 0 {
						info.SubnetMask = adapterConfigs[0].IPSubnet[0]
					}
					
					// 获取网关
					if len(adapterConfigs[0].DefaultIPGateway) > 0 {
						info.Gateway = adapterConfigs[0].DefaultIPGateway[0]
					}
					
					// 获取DNS服务器
					info.DNSServers = adapterConfigs[0].DNSServerSearchOrder
					
					// 获取IP获取方式
					if adapterConfigs[0].DHCPEnabled {
						info.IPAcquisitionMode = "DHCP"
					} else {
						info.IPAcquisitionMode = "静态IP"
					}
					
					// 获取MAC地址（如果前面没有获取到）
					if info.MacAddress == "" {
						info.MacAddress = adapterConfigs[0].MACAddress
					}
				}
			}
		}
	}

	// 使用netsh wlan show interfaces获取更详细的无线网络信息
	if strings.Contains(strings.ToLower(info.InterfaceName), "wi-fi") || 
	   strings.Contains(strings.ToLower(info.InterfaceName), "wireless") || 
	   strings.Contains(strings.ToLower(info.InterfaceName), "wlan") {
		cmd := exec.Command("netsh", "wlan", "show", "interfaces")
		wlanOutput, err := cmd.Output()
		if err == nil {
			wlanOutputStr := string(wlanOutput)
			log.Printf("获取到WiFi接口信息: %s", wlanOutputStr)
			
			// 获取SSID
			ssidRegex := regexp.MustCompile(`SSID\s*:\s*(.+)`)
			ssidMatches := ssidRegex.FindStringSubmatch(wlanOutputStr)
			if len(ssidMatches) > 1 {
				info.WiFi.SSID = strings.TrimSpace(ssidMatches[1])
				log.Printf("解析到SSID: %s", info.WiFi.SSID)
			}
			
			// 获取BSSID
			bssidRegex := regexp.MustCompile(`BSSID\s*:\s*(.+)`)
			bssidMatches := bssidRegex.FindStringSubmatch(wlanOutputStr)
			if len(bssidMatches) > 1 {
				info.WiFi.BSSID = strings.TrimSpace(bssidMatches[1])
				log.Printf("解析到BSSID: %s", info.WiFi.BSSID)
			}
			
			// 获取信号强度
			signalRegex := regexp.MustCompile(`信号\s*:\s*(\d+)%`)
			if !strings.Contains(wlanOutputStr, "信号") {
				// 英文系统
				signalRegex = regexp.MustCompile(`Signal\s*:\s*(\d+)%`)
			}
			signalMatches := signalRegex.FindStringSubmatch(wlanOutputStr)
			if len(signalMatches) > 1 {
				signalStr := strings.TrimSpace(signalMatches[1])
				signalInt, _ := strconv.Atoi(signalStr)
				// 转换为dBm (近似值，信号百分比转dBm)
				// 假设100%约等于-30dBm，0%约等于-100dBm
				signalDBM := -100 + (signalInt * 70 / 100)
				info.WiFi.RSSI = signalDBM
				info.WiFi.SignalStrength = signalDBM
				log.Printf("解析到信号强度: %d%%, 转换为RSSI: %d dBm", signalInt, signalDBM)
			}
			
			// 设置默认噪声值
			info.WiFi.Noise = -90 // 默认噪声值为-90dBm
			log.Printf("设置默认噪声值: %d dBm", info.WiFi.Noise)
			
			// 获取频道
			channelRegex := regexp.MustCompile(`信道\s*:\s*(\d+)`)
			if !strings.Contains(wlanOutputStr, "信道") {
				// 英文系统
				channelRegex = regexp.MustCompile(`Channel\s*:\s*(\d+)`)
			}
			channelMatches := channelRegex.FindStringSubmatch(wlanOutputStr)
			if len(channelMatches) > 1 {
				channel := strings.TrimSpace(channelMatches[1])
				channelInt, _ := strconv.Atoi(channel)
				info.WiFi.Channel = channelInt
				log.Printf("解析到频道: %d", info.WiFi.Channel)
				
				// 设置频率 (根据频道计算)
				if channelInt > 0 {
					if channelInt <= 14 {
						// 2.4 GHz频段
						info.WiFi.Frequency = 2.412 + 0.005*float64(channelInt-1)
					} else {
						// 5 GHz频段
						info.WiFi.Frequency = 5.0 + 0.005*float64(channelInt)
					}
					log.Printf("计算频率: %.3f GHz", info.WiFi.Frequency)
				}
			}
			
			// 获取传输速率
			txRateRegex := regexp.MustCompile(`传输速率.+?:\s*(\d+\.?\d*)\s*(Mbps|Gbps)`)
			if !strings.Contains(wlanOutputStr, "传输速率") {
				// 英文系统
				txRateRegex = regexp.MustCompile(`Transmit Rate.+?:\s*(\d+\.?\d*)\s*(Mbps|Gbps)`)
			}
			txRateMatches := txRateRegex.FindStringSubmatch(wlanOutputStr)
			if len(txRateMatches) > 2 {
				txRate := strings.TrimSpace(txRateMatches[1])
				unit := strings.TrimSpace(txRateMatches[2])
				txRateFloat, _ := strconv.ParseFloat(txRate, 64)
				if unit == "Gbps" {
					txRateFloat *= 1000 // 转换为Mbps
				}
				info.WiFi.TxRate = int(txRateFloat)
				log.Printf("解析到传输速率: %d Mbps", info.WiFi.TxRate)
				
				// 根据PHY模式和传输速率估算MCS索引和NSS
				txRateNum := info.WiFi.TxRate
				if strings.Contains(info.WiFi.PHYMode, "802.11ax") {
					// 802.11ax (Wi-Fi 6)
					if txRateNum > 1200 {
						info.WiFi.MCS = 11
						info.WiFi.NSS = 4
						log.Printf("估算802.11ax MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					} else if txRateNum > 600 {
						info.WiFi.MCS = 11
						info.WiFi.NSS = 2
						log.Printf("估算802.11ax MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					} else {
						info.WiFi.MCS = 9
						info.WiFi.NSS = 1
						log.Printf("估算802.11ax MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					}
				} else if strings.Contains(info.WiFi.PHYMode, "802.11ac") {
					// 802.11ac (Wi-Fi 5)
					if txRateNum > 800 {
						info.WiFi.MCS = 9
						info.WiFi.NSS = 3
						log.Printf("估算802.11ac MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					} else if txRateNum > 400 {
						info.WiFi.MCS = 9
						info.WiFi.NSS = 2
						log.Printf("估算802.11ac MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					} else {
						info.WiFi.MCS = 7
						info.WiFi.NSS = 1
						log.Printf("估算802.11ac MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					}
				} else {
					// 802.11n或更早
					if txRateNum > 150 {
						info.WiFi.MCS = 7
						info.WiFi.NSS = 2
						log.Printf("估算802.11n MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					} else {
						info.WiFi.MCS = 7
						info.WiFi.NSS = 1
						log.Printf("估算802.11n MCS索引: %d, NSS: %d", info.WiFi.MCS, info.WiFi.NSS)
					}
				}
			}
			
			// 获取无线模式
			modeRegex := regexp.MustCompile(`无线电类型\s*:\s*(.+)`)
			if !strings.Contains(wlanOutputStr, "无线电类型") {
				// 英文系统
				modeRegex = regexp.MustCompile(`Radio type\s*:\s*(.+)`)
			}
			modeMatches := modeRegex.FindStringSubmatch(wlanOutputStr)
			if len(modeMatches) > 1 {
				mode := strings.TrimSpace(modeMatches[1])
				info.WiFi.PHYMode = mode
				
				// 设置支持的PHY模式
				if strings.Contains(mode, "802.11ax") {
					info.WiFi.SupportedPHY = "802.11 a/b/g/n/ac/ax"
				} else if strings.Contains(mode, "802.11ac") {
					info.WiFi.SupportedPHY = "802.11 a/b/g/n/ac"
				} else if strings.Contains(mode, "802.11n") {
					info.WiFi.SupportedPHY = "802.11 a/b/g/n"
				} else {
					info.WiFi.SupportedPHY = "802.11 a/b/g"
				}
				
				// 估算MCS索引和NSS (空间流数量)
				if info.WiFi.TxRate > 0 {
					// 根据传输速率和PHY模式估算MCS和NSS
					if strings.Contains(mode, "802.11ax") {
						// 802.11ax (Wi-Fi 6)
						if info.WiFi.TxRate > 1200 {
							info.WiFi.MCS = 11
							info.WiFi.NSS = 4
						} else if info.WiFi.TxRate > 600 {
							info.WiFi.MCS = 11
							info.WiFi.NSS = 2
						} else {
							info.WiFi.MCS = 9
							info.WiFi.NSS = 1
						}
					} else if strings.Contains(mode, "802.11ac") {
						// 802.11ac (Wi-Fi 5)
						if info.WiFi.TxRate > 800 {
							info.WiFi.MCS = 9
							info.WiFi.NSS = 3
						} else if info.WiFi.TxRate > 400 {
							info.WiFi.MCS = 9
							info.WiFi.NSS = 2
						} else {
							info.WiFi.MCS = 7
							info.WiFi.NSS = 1
						}
					} else {
						// 802.11n或更早
						if info.WiFi.TxRate > 150 {
							info.WiFi.MCS = 7
							info.WiFi.NSS = 2
						} else {
							info.WiFi.MCS = 7
							info.WiFi.NSS = 1
						}
					}
				}
			}
			
			// 获取认证方式
			authRegex := regexp.MustCompile(`身份验证\s*:\s*(.+)`)
			if !strings.Contains(wlanOutputStr, "身份验证") {
				// 英文系统
				authRegex = regexp.MustCompile(`Authentication\s*:\s*(.+)`)
			}
			authMatches := authRegex.FindStringSubmatch(wlanOutputStr)
			if len(authMatches) > 1 {
				auth := strings.TrimSpace(authMatches[1])
				info.WiFi.AuthMethod = auth
			}
			
			// 获取国家/地区代码
			countryRegex := regexp.MustCompile(`国家/地区代码\s*:\s*(.+)`)
			if !strings.Contains(wlanOutputStr, "国家/地区代码") {
				// 英文系统
				countryRegex = regexp.MustCompile(`Country/Region code\s*:\s*(.+)`)
			}
			countryMatches := countryRegex.FindStringSubmatch(wlanOutputStr)
			if len(countryMatches) > 1 {
				country := strings.TrimSpace(countryMatches[1])
				info.WiFi.CountryCode = country
				info.CountryCode = country
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

	// 中英文系统的IPv4路由表标识
	ipv4Headers := []string{"IPv4 Route Table", "IPv4 路由表"}
	ipv6Headers := []string{"IPv6 Route Table", "IPv6 路由表"}
	tableHeaders := []string{"Network Destination", "网络目标"}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 识别IPv4路由表部分
		for _, header := range ipv4Headers {
			if strings.Contains(line, header) {
				inIPv4Section = true
				break
			}
		}
		if inIPv4Section {
			// 识别IPv6路由表部分（结束IPv4部分）
			for _, header := range ipv6Headers {
				if strings.Contains(line, header) {
					inIPv4Section = false
					break
				}
			}
		}
		
		if !inIPv4Section {
			continue
		}

		// 跳过空行
		if line == "" {
			continue
		}

		// 识别表头行
		for _, header := range tableHeaders {
			if strings.Contains(line, header) {
				headerFound = true
				break
			}
		}
		if !headerFound {
			continue
		}

		// 解析路由条目
		fields := regexp.MustCompile(`\s+`).Split(line, -1)
		if len(fields) >= 5 {
			// 过滤掉表头行和空字段
			if fields[0] == "Network" || fields[0] == "网络" || fields[0] == "" {
				continue
			}
			
			routes = append(routes, model.RouteEntry{
				Destination: fields[0],
				Gateway:     fields[1],
				Flags:       fields[3], // 使用Metric作为Flags
				Interface:   fields[4],
				Netmask:     fields[2], // 使用Genmask作为Netmask
			})
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
	log.Printf("获取到WiFi接口信息: %s", outputStr)

	// 提取SSID
	ssidRegex := regexp.MustCompile(`SSID\s+:\s+(.+)`)
	ssidMatches := ssidRegex.FindStringSubmatch(outputStr)
	if len(ssidMatches) > 1 {
		wifiInfo.SSID = strings.TrimSpace(ssidMatches[1])
		log.Printf("解析到SSID: %s", wifiInfo.SSID)
	}

	// 提取BSSID
	bssidRegex := regexp.MustCompile(`BSSID\s+:\s+(.+)`)
	bssidMatches := bssidRegex.FindStringSubmatch(outputStr)
	if len(bssidMatches) > 1 {
		wifiInfo.BSSID = strings.TrimSpace(bssidMatches[1])
		log.Printf("解析到BSSID: %s", wifiInfo.BSSID)
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

		// 设置噪声值（Windows没有直接获取噪声的方法，使用估算值）
		// 噪声通常在-95dBm到-85dBm之间
		wifiInfo.Noise = -90 // 使用一个合理的默认值
		log.Printf("解析到信号强度: %d%%, 转换为RSSI: %d dBm", signal, rssi)
		log.Printf("设置默认噪声值: %d dBm", wifiInfo.Noise)
	}

	// 提取频道
	channelRegex := regexp.MustCompile(`Channel\s+:\s+(\d+)`)
	channelMatches := channelRegex.FindStringSubmatch(outputStr)
	if len(channelMatches) > 1 {
		channel := strings.TrimSpace(channelMatches[1])
		channelNum, _ := strconv.Atoi(channel)
		wifiInfo.Channel = channelNum
		log.Printf("解析到频道: %d", wifiInfo.Channel)

		// 设置频率（根据频道计算）
		if channelNum > 0 {
			if channelNum <= 14 {
				// 2.4 GHz频段
				wifiInfo.Frequency = 2.412 + 0.005*float64(channelNum-1)
			} else {
				// 5 GHz频段
				wifiInfo.Frequency = 5.0 + 0.005*float64(channelNum)
			}
			log.Printf("计算频率: %.3f GHz", wifiInfo.Frequency)
		}
	}

	// 提取PHY模式
	radioTypeRegex := regexp.MustCompile(`Radio type\s+:\s+(.+)`)
	radioTypeMatches := radioTypeRegex.FindStringSubmatch(outputStr)
	if len(radioTypeMatches) > 1 {
		radioType := strings.TrimSpace(radioTypeMatches[1])

		// 将Windows的无线电类型映射到PHY模式
		phyModeMap := map[string]string{
			"802.11n":  "802.11n",
			"802.11ac": "802.11ac",
			"802.11ax": "802.11ax",
			"802.11a":  "802.11a",
			"802.11g":  "802.11g",
			"802.11b":  "802.11b",
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

	// 获取传输速率
	txRateRegex := regexp.MustCompile(`Transmit\s+rate\s+\(Mbps\)\s+:\s+(\d+\.?\d*)\s*(Mbps|Gbps)`)
	txRateMatches := txRateRegex.FindStringSubmatch(outputStr)
	if len(txRateMatches) > 2 {
		txRate := strings.TrimSpace(txRateMatches[1])
		unit := strings.TrimSpace(txRateMatches[2])
		txRateFloat, _ := strconv.ParseFloat(txRate, 64)
		if unit == "Gbps" {
			txRateFloat *= 1000 // 转换为Mbps
		}
		wifiInfo.TxRate = int(txRateFloat)
		log.Printf("解析到传输速率: %d Mbps", wifiInfo.TxRate)
		
		// 根据PHY模式和传输速率估算MCS索引和NSS
		txRateNum := wifiInfo.TxRate
		if strings.Contains(wifiInfo.PHYMode, "802.11ax") {
			// 802.11ax (Wi-Fi 6)
			if txRateNum > 1200 {
				wifiInfo.MCS = 11
				wifiInfo.NSS = 4
				log.Printf("估算802.11ax MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			} else if txRateNum > 600 {
				wifiInfo.MCS = 11
				wifiInfo.NSS = 2
				log.Printf("估算802.11ax MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			} else {
				wifiInfo.MCS = 9
				wifiInfo.NSS = 1
				log.Printf("估算802.11ax MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			}
		} else if strings.Contains(wifiInfo.PHYMode, "802.11ac") {
			// 802.11ac (Wi-Fi 5)
			if txRateNum > 800 {
				wifiInfo.MCS = 9
				wifiInfo.NSS = 3
				log.Printf("估算802.11ac MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			} else if txRateNum > 400 {
				wifiInfo.MCS = 9
				wifiInfo.NSS = 2
				log.Printf("估算802.11ac MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			} else {
				wifiInfo.MCS = 7
				wifiInfo.NSS = 1
				log.Printf("估算802.11ac MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			}
		} else {
			// 802.11n或更早
			if txRateNum > 150 {
				wifiInfo.MCS = 7
				wifiInfo.NSS = 2
				log.Printf("估算802.11n MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
			} else {
				wifiInfo.MCS = 7
				wifiInfo.NSS = 1
				log.Printf("估算802.11n MCS索引: %d, NSS: %d", wifiInfo.MCS, wifiInfo.NSS)
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

	// 如果连接了WiFi，设置IsConnected为true
	if wifiInfo.SSID != "" {
		wifiInfo.IsConnected = true
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
