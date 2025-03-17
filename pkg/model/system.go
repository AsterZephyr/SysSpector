package model

// SystemInfo 表示收集的系统信息的总体结构
type SystemInfo struct {
	Hostname     string      // 计算机主机名
	OS           string      // 操作系统类型和版本
	Model        string      // 设备型号
	SerialNumber string      // 设备序列号
	CPU          CPUInfo     // CPU信息
	Memory       MemoryInfo  // 内存信息
	Disks        []Disk      // 磁盘信息列表
	UUID         string      // 硬件UUID（Windows为UUID，macOS为BRUUID）
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
