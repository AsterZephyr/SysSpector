package model

// SystemInfo 表示收集的系统信息的总体结构
type SystemInfo struct {
	Hostname     string     // 计算机主机名
	OS           string     // 操作系统类型和版本
	Model        string     // 设备型号
	ModelID      string     // 设备型号标识符
	SerialNumber string     // 设备序列号
	UUID         string     // 硬件UUID（Windows为UUID，macOS为BRUUID）
	CPU          CPUInfo    // CPU信息
	Memory       MemoryInfo // 内存信息
	Disks        []Disk     // 磁盘信息列表

	// 新增硬件动态数据
	DiskUsage    []DiskPartitionInfo // 硬盘使用情况
	MemoryUsage  MemoryUsageInfo     // 内存使用情况
	Battery      BatteryInfo         // 电池信息
	ACAdapter    ACAdapterInfo       // 交流充电器信息
	Bluetooth    BluetoothInfo       // 蓝牙信息
	Temperature  []TempSensorInfo    // 设备温度信息
	WiFiAutoJoin WiFiAutoJoinInfo    // WiFi自动连接状态

	// 新增网络客户端动态数据
	Network NetworkInfo // 网络信息

	// 新增系统信息
	SystemVersion string // 系统版本
	ComputerName  string // 电脑名称
	UpTime        string // 启动后的时间长度

	// 新增软件信息
	InstalledApps []AppInfo     // 已安装应用信息
	RunningApps   []ProcessInfo // 正在运行的应用信息
}

// CPUInfo 表示处理器信息
type CPUInfo struct {
	Model string // 处理器型号名称
	Cores int    // 处理器核心数
}

// MemoryInfo 表示内存信息
type MemoryInfo struct {
	Total uint64 // 总内存容量（字节）
	Type  string // 内存类型（如LPDDR5, DDR4等）
}

// Disk 表示存储设备信息
type Disk struct {
	Name   string // 设备名称
	Size   uint64 // 容量（GB）
	Serial string // 序列号
	Model  string // 设备型号
}

// DiskUsageInfo 表示硬盘使用情况
type DiskUsageInfo struct {
	Total      uint64              // 总容量（字节）
	Used       uint64              // 已用容量（字节）
	Free       uint64              // 可用容量（字节）
	UsedPerc   float64             // 使用百分比
	Partitions []DiskPartitionInfo // 分区信息
}

// DiskPartitionInfo 表示硬盘分区信息
type DiskPartitionInfo struct {
	MountPoint  string  // 挂载点
	Filesystem  string  // 文件系统类型
	Total       uint64  // 总容量（字节）
	Used        uint64  // 已用容量（字节）
	Free        uint64  // 可用容量（字节）
	UsedPerc    float64 // 使用百分比
}

// MemoryUsageInfo 表示内存使用情况
type MemoryUsageInfo struct {
	Total    uint64  // 总容量（字节）
	Used     uint64  // 已用容量（字节）
	Free     uint64  // 可用容量（字节）
	UsedPerc float64 // 使用百分比
	Active   uint64  // 活跃内存（字节）
	Inactive uint64  // 不活跃内存（字节）
	Cached   uint64  // 已缓存内存（字节）
}

// BatteryInfo 表示电池信息
type BatteryInfo struct {
	Percentage    int    // 电量百分比
	IsCharging    bool   // 是否正在充电
	IsPresent     bool   // 是否存在电池
	CycleCount    int    // 电池循环计数
	Health        string // 电池健康状态
	Status        string // 电池状态
	TimeRemaining int    // 剩余使用时间（分钟）
}

// ACAdapterInfo 表示交流充电器信息
type ACAdapterInfo struct {
	Connected   bool   // 是否连接
	IsConnected bool   // 是否连接（兼容性字段）
	SerialNum   string // 序列号
	Name        string // 名称
	Wattage     int    // 功率（瓦）
}

// BluetoothInfo 表示蓝牙信息
type BluetoothInfo struct {
	Enabled          bool           // 是否启用
	IsAvailable      bool           // 是否可用
	Status           string         // 状态
	Name             string         // 名称
	Address          string         // 地址
	Devices          []BTDeviceInfo // 已连接设备列表
	ConnectedDevices []BTDeviceInfo // 已连接设备列表（兼容性字段）
}

// BTDeviceInfo 表示蓝牙设备信息
type BTDeviceInfo struct {
	Name      string // 设备名称
	Address   string // 设备地址
	Type      string // 设备类型
	Connected bool   // 是否已连接
}

// TemperatureInfo 表示设备温度信息
type TemperatureInfo struct {
	CPU     float64          // CPU温度（摄氏度）
	GPU     float64          // GPU温度（摄氏度）
	Sensors []TempSensorInfo // 温度传感器信息
}

// TempSensorInfo 表示温度传感器信息
type TempSensorInfo struct {
	Name        string  // 传感器名称
	Temperature float64 // 温度（摄氏度）
	Location    string  // 位置
	Sensor      string  // 传感器名称（兼容性字段）
	Value       float64 // 温度值（兼容性字段）
}

// NetworkInfo 表示网络信息
type NetworkInfo struct {
	WiFi        WiFiInfo           // WiFi信息
	IP          string             // IP地址
	MacAddress  string             // MAC地址
	AWDLStatus  string             // AWDL状态
	AWDLEnabled bool               // AWDL是否启用
	Interfaces  []NetInterfaceInfo // 网络接口信息
	Traffic     TrafficInfo        // 网络流量
	Latency     LatencyInfo        // 网络延迟
	VPN         VPNInfo            // VPN信息
	DNS         DNSConfigInfo      // DNS配置
	DNSServers  []string           // DNS服务器列表（兼容性字段）
	PublicIP    string             // 公网IP
	ProxyStatus bool               // 网络代理状态
	ProxyInfo   ProxyInfo          // 代理信息
	AvgLatency  float64            // 平均延迟（毫秒）
}

// WiFiInfo 表示WiFi信息
type WiFiInfo struct {
	SSID           string  // SSID
	BSSID          string  // BSSID
	CountryCode    string  // 国家/地区代码
	RSSI           int     // 信号强度
	Noise          int     // 噪声
	PHYMode        string  // PHY模式
	Channel        int     // 频道
	TxRate         int     // 传输速率
	MSCIndex       int     // MSC索引
	NSS            int     // NSS
	SupportedPHY   string  // 支持的PHY模式
	IsConnected    bool    // 是否已连接
	SignalStrength int     // 信号强度（dBm）
	Frequency      float64 // 频率（GHz）
}

// NetInterfaceInfo 表示网络接口信息
type NetInterfaceInfo struct {
	Name        string   // 接口名称
	IP          string   // IP地址
	MacAddress  string   // MAC地址
	MACAddress  string   // MAC地址（兼容性字段）
	IsUp        bool     // 是否启用
	Status      string   // 状态
	IPAddresses []string // IP地址列表
	InBytes     uint64   // 接收的字节数
	OutBytes    uint64   // 发送的字节数
}

// TrafficInfo 表示网络流量信息
type TrafficInfo struct {
	BytesSent      uint64               // 发送字节数
	BytesReceived  uint64               // 接收字节数
	PacketsSent    uint64               // 发送数据包数
	PacketsRecv    uint64               // 接收数据包数
	ProcessTraffic []ProcessTrafficInfo // 各进程流量
}

// ProcessTrafficInfo 表示进程网络流量信息
type ProcessTrafficInfo struct {
	PID           int                // 进程ID
	Name          string             // 进程名称
	ProcessName   string             // 进程名称（兼容性字段）
	BytesIn       uint64             // 接收字节数
	BytesOut      uint64             // 发送字节数
	Connections   []ConnectionInfo   // 连接信息
	ConnectionCount int              // 连接数量
}

// ConnectionInfo 表示网络连接信息
type ConnectionInfo struct {
	LocalAddr  string // 本地地址
	LocalPort  int    // 本地端口
	RemoteAddr string // 远程地址
	RemotePort int    // 远程端口
	Protocol   string // 协议
	State      string // 连接状态
}

// LatencyInfo 表示网络延迟信息
type LatencyInfo struct {
	AvgLatency  float64             // 平均延迟（毫秒）
	Jitter      float64             // 抖动（毫秒）
	PacketLoss  float64             // 丢包率（百分比）
	Targets     []TargetLatencyInfo // 目标延迟信息
	NetworkHops []NetworkHopInfo    // 网络跳点信息
}

// TargetLatencyInfo 表示目标延迟信息
type TargetLatencyInfo struct {
	TargetName  string  // 目标名称
	TargetHost  string  // 目标主机
	MinLatency  float64 // 最小延迟（毫秒）
	AvgLatency  float64 // 平均延迟（毫秒）
	MaxLatency  float64 // 最大延迟（毫秒）
	StdDev      float64 // 标准差（毫秒）
	PacketLoss  float64 // 丢包率（百分比）
	Jitter      float64 // 抖动（毫秒）
}

// NetworkHopInfo 表示网络跳点信息
type NetworkHopInfo struct {
	HopNum      int     // 跳点序号
	Host        string  // 主机地址
	Loss        float64 // 丢包率（百分比）
	SentPackets int     // 发送的数据包数
	LastLatency float64 // 最后一次延迟（毫秒）
	AvgLatency  float64 // 平均延迟（毫秒）
	BestLatency float64 // 最佳延迟（毫秒）
	WorstLatency float64 // 最差延迟（毫秒）
	StdDev      float64 // 标准差（毫秒）
}

// VPNInfo 表示VPN信息
type VPNInfo struct {
	Connected        bool          // 是否已连接
	IsConnected      bool          // 是否已连接（兼容性字段）
	NodeName         string        // 节点名称
	Provider         string        // 提供商
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

// NetworkTrafficInfo 表示网络流量信息
type NetworkTrafficInfo struct {
	Interface     string // 接口名称
	BytesReceived uint64 // 接收字节数
	BytesSent     uint64 // 发送字节数
	PacketsReceived uint64 // 接收数据包数
	PacketsSent   uint64 // 发送数据包数
}

// WiFiAutoJoinInfo 表示WiFi自动连接信息
type WiFiAutoJoinInfo struct {
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

// AppInfo 表示应用信息
type AppInfo struct {
	Name        string // 应用名称
	Version     string // 版本
	InstallDate string // 安装日期
	Path        string // 安装路径
}

// ProcessInfo 表示进程信息
type ProcessInfo struct {
	PID          int     // 进程ID
	Name         string  // 进程名称
	CPU          float64 // CPU使用率
	Memory       uint64  // 内存使用量（字节）
	NetworkUsage uint64  // 网络使用量（字节/秒）
}
