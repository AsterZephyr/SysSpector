package model

// SystemInfo represents the collected system information
type SystemInfo struct {
	Hostname     string
	OS           string
	Model        string   // Device model (e.g., MacBookPro18,3)
	SerialNumber string   // Serial number
	CPU          CPUInfo
	Memory       MemoryInfo
	Disks        []Disk
	UUID         string   // Windows UUID or macOS BRUUID
}

// CPUInfo contains processor information
type CPUInfo struct {
	Model string
	Cores int
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total uint64 // Total memory in bytes
	Type  string // Memory type (e.g., LPDDR5)
}

// Disk represents storage device information
type Disk struct {
	Name   string
	Size   uint64
	Serial string
	Model  string
}
