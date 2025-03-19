package model

// SystemInfo 表示收集的系统信息的总体结构
type SystemInfo struct {
	Hostname      string
	OS            string
	Model         string
	ModelID       string
	SerialNumber  string
	UUID          string
	CPU           CPUInfo
	Memory        MemoryInfo
	Disks         []Disk
	DiskUsage     []DiskPartitionInfo
	MemoryUsage   MemoryUsageInfo
	Battery       BatteryInfo
	ACAdapter     ACAdapterInfo
	Bluetooth     BluetoothInfo
	Temperature   []TempSensorInfo
	Network       NetworkInfo      // 网络信息
	WiFiAutoJoin  WiFiAutoJoinInfo // WiFi自动连接状态
	SystemVersion string
	ComputerName  string
	UpTime        string
	InstalledApps []AppInfo
	RunningApps   []ProcessInfo
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

// DiskPartitionInfo 表示硬盘分区信息
type DiskPartitionInfo struct {
	MountPoint string  // 挂载点
	Filesystem string  // 文件系统类型
	Total      uint64  // 总容量（字节）
	Used       uint64  // 已用容量（字节）
	Free       uint64  // 可用容量（字节）
	UsedPerc   float64 // 使用百分比
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
	ConnectedDevices []BTDeviceInfo // 已连接设备列表
	Devices          []BTDeviceInfo // 已连接设备列表（兼容性字段）
}

// BTDeviceInfo 表示蓝牙设备信息
type BTDeviceInfo struct {
	Name      string // 设备名称
	Address   string // 设备地址
	Type      string // 设备类型
	Connected bool   // 是否已连接
}

// TempSensorInfo 表示温度传感器信息
type TempSensorInfo struct {
	Name        string  // 传感器名称
	Temperature float64 // 温度（摄氏度）
	Location    string  // 位置
	Sensor      string  // 传感器名称（兼容性字段）
	Value       float64 // 温度值（兼容性字段）
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

