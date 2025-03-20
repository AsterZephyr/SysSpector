package model

// NetworkInfo 表示网络信息
type NetworkInfo struct {
	// WiFi信息
	WiFi WiFiInfo

	// 客户端信息
	IP         string // 客户端IP地址
	MacAddress string // 客户端MAC地址

	// 国家/地区代码
	CountryCode string // 用户当前所在地区代码

	// AWDL信息
	AWDLStatus  string // AWDL状态
	AWDLEnabled bool   // AWDL是否启用

	// 公网IP信息
	PublicIP string // 公网出口IP

	// DNS信息
	DNS        DNSConfigInfo
	DNSServers []string // DNS服务器列表（兼容性字段）

	// VPN信息
	VPN VPNInfo

	// 网络延迟信息
	Latency LatencyInfo

	// 网络代理状态
	ProxyStatus bool      // 网络代理是否开启
	ProxyInfo   ProxyInfo // 代理信息

	// 客户端路由表
	RouteTable []RouteEntry // 路由表条目

	// 网卡流量
	NetworkTraffic string // 网卡流量（KB/s）

	// 各进程流量
	ProcessTraffic string // 各进程流量（KB/s）
}

// WiFiInfo 表示WiFi信息
type WiFiInfo struct {
	SSID           string  // WiFi网络名称
	BSSID          string  // WiFi基站MAC地址
	IsConnected    bool    // 是否已连接WiFi
	SignalStrength int     // 信号强度（dBm）
	RSSI           int     // 接收信号强度指示（dBm）
	Noise          int     // 噪声（dBm）
	Channel        int     // 频道
	Frequency      float64 // 频率（GHz）
	PHYMode        string  // 物理层模式（如802.11ac）
	TxRate         int     // 传输速率（Mbps）
	MCS            int     // MCS索引
	NSS            int     // 空间流数量
	CountryCode    string  // WiFi国家/地区代码
	SupportedPHY   string  // 支持的PHY模式
}

// DNSConfigInfo 表示DNS配置信息
type DNSConfigInfo struct {
	Servers         []string    // DNS服务器列表
	SearchDomains   []string    // 搜索域列表
	ResolutionOrder []string    // 解析顺序
	HostsFile       string      // hosts文件内容
	ResolvConfFile  string      // resolv.conf文件内容
	HostEntries     []HostEntry // hosts条目
}

// HostEntry 表示hosts文件中的条目
type HostEntry struct {
	IP       string // IP地址
	Hostname string // 主机名
}

// DNSInfo 表示DNS信息（兼容性结构体）
type DNSInfo struct {
	Servers []string // DNS服务器列表
}

// VPNInfo 表示VPN信息
type VPNInfo struct {
	IsConnected      bool          // 是否已连接VPN
	Provider         string        // VPN提供商
	NodeName         string        // VPN节点名称
	Services         []string      // 服务列表
	Nodes            []string      // 节点列表
	Server           string        // 服务器
	Status           string        // 状态
	ActiveConnection string        // 活动连接
	ConnectionID     string        // 连接ID
	Interfaces       []string      // 接口列表
	NodeInfos        []VPNNodeInfo // 节点详细信息
	ConfigFile       string        // 配置文件路径
}

// VPNNodeInfo 表示VPN节点信息
type VPNNodeInfo struct {
	Name   string // 节点名称
	ID     string // 节点ID
	Status string // 节点状态
}

// LatencyInfo 表示网络延迟信息
type LatencyInfo struct {
	AvgLatency  float64             // 平均延迟（ms）
	Targets     []TargetLatencyInfo // 延迟目标列表
	NetworkHops []NetworkHopInfo    // 网络跳点信息
	Jitter      float64             // 抖动（毫秒）
	PacketLoss  float64             // 丢包率（百分比）
}

// TargetLatencyInfo 表示目标延迟信息
type TargetLatencyInfo struct {
	TargetName string  // 目标名称
	TargetHost string  // 目标主机
	MinLatency float64 // 最小延迟（ms）
	AvgLatency float64 // 平均延迟（ms）
	MaxLatency float64 // 最大延迟（ms）
	PacketLoss float64 // 丢包率（%）
	StdDev     float64 // 标准差（毫秒）
	Jitter     float64 // 抖动（毫秒）
}

// NetworkHopInfo 表示网络跳点信息
type NetworkHopInfo struct {
	HopNum       int     // 跳点序号
	Host         string  // 主机地址
	Loss         float64 // 丢包率（百分比）
	SentPackets  int     // 发送的数据包数
	LastLatency  float64 // 最后一次延迟（毫秒）
	AvgLatency   float64 // 平均延迟（毫秒）
	BestLatency  float64 // 最佳延迟（毫秒）
	WorstLatency float64 // 最差延迟（毫秒）
	StdDev       float64 // 标准差（毫秒）
}

// WiFiAutoJoinInfo 表示WiFi自动连接状态
type WiFiAutoJoinInfo struct {
	Enabled      bool              // 是否启用自动连接
	IsConfigured bool              // 是否配置
	Status       string            // 状态
	Networks     []WiFiNetworkInfo // 网络列表
}

// WiFiNetworkInfo 表示WiFi网络信息
type WiFiNetworkInfo struct {
	SSID     string // 网络名称
	AutoJoin bool   // 是否自动连接
}

// ProxyInfo 表示代理信息
type ProxyInfo struct {
	Enabled bool   // 是否启用
	Server  string // 服务器地址
	Port    int    // 端口
}

// RouteEntry 表示路由表条目
type RouteEntry struct {
	Destination string // 目标地址
	Gateway     string // 网关
	Flags       string // 标志
	Interface   string // 接口
	Netmask     string // 子网掩码
}
