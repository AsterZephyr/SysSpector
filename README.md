# SysSpector

SysSpector is a cross-platform system information collection tool written in Go. It supports both Windows and macOS platforms and collects detailed hardware and system information.

## Features

- Collects system information including:
  - Computer name and operating system type
  - Hardware model and serial number
  - Processor name and core count
  - Memory capacity and type (e.g., LPDDR5)
  - Storage device information (model, capacity, serial number)
  - Special identifiers (Windows UUID, macOS BRUUID)
- Cross-platform support for Windows and macOS
- JSON output format for easy parsing and integration

## Installation

### Prerequisites

- Go 1.16 or higher

### Building from source

```bash
# Clone the repository
git clone https://github.com/hexingze/SysSpector.git
cd SysSpector

# Build the application
go build -o sysinfo ./cmd/sysinfo
```

## Usage

Run the application to collect and display system information:

```bash
./sysinfo
```

Save the output to a file:

```bash
./sysinfo --save output.json
```

## Technical Implementation

### Cross-platform Architecture

- Uses `runtime.GOOS` to detect the operating system
- Platform-specific implementations:
  - Windows: Uses WMI interface to query hardware information
  - macOS: Uses system commands (sysctl, system_profiler, ioreg) to collect information
- Common information collection using gopsutil library

### Dependencies

- [github.com/shirou/gopsutil/v3](https://github.com/shirou/gopsutil) - Cross-platform hardware monitoring
- [github.com/StackExchange/wmi](https://github.com/StackExchange/wmi) - Windows WMI queries

## Project Structure

```
SysSpector/
├── cmd/                  // Main program entry
│   └── sysinfo/
│       └── main.go       // Initialization and output logic
├── internal/             // Platform-specific implementations
│   ├── windows/          
│   │   └── wmi.go        // WMI query wrapper
│   └── darwin/
│       └── command.go    // macOS command parsing
├── pkg/                  // Public libraries
│   └── model/
│       └── system.go     // Data model definitions
├── go.mod
└── README.md             // Project documentation
```

## License

MIT
