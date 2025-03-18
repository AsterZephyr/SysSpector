# SysSpector

SysSpector 是一个跨平台的系统信息收集工具，支持 Windows 和 macOS 平台。它收集详细的硬件和系统信息。

## 功能特点

- 收集系统信息，包括：
  - 主机名和操作系统类型
  - 硬件型号和序列号
  - 处理器名称和核心数
  - 内存容量和类型（例如 LPDDR5）
  - 存储设备信息（型号、容量、序列号）
  - 特殊标识符（Windows UUID、macOS BRUUID）
- 支持 Windows 和 macOS 平台
- JSON 输出格式，方便解析和集成

## 安装

### 前置条件

- Go 1.16 或更高版本

### 从源代码构建

```bash
# 克隆仓库
git clone https://github.com/hexingze/SysSpector.git
cd SysSpector

# 构建应用程序
go build -o sysinfo ./cmd/sysinfo
```

## 使用方法

运行应用程序以收集和显示系统信息：

```bash
./sysinfo
```

保存输出到文件：

```bash
./sysinfo --save output.json
```

## 技术实现

### 跨平台架构

- 使用 `runtime.GOOS` 检测操作系统
- 平台特定实现：
  - Windows：使用 WMI 接口查询硬件信息
  - macOS：使用系统命令（sysctl、system_profiler、ioreg）收集信息
- 公共信息收集使用 gopsutil 库

### 平台特定说明

#### Windows

SysSpector 在 Windows 上使用 WMI (Windows Management Instrumentation) 收集系统信息。由于不同 Windows 版本的 WMI 实现可能有所不同，因此在某些 Windows 版本上可能会遇到兼容性问题。

我们已经实现了以下机制来处理这些兼容性问题：

1. 错误处理：对 WMI 查询错误进行详细日志记录，以便于调试
2. 备选查询：为关键信息提供备选查询方式，以适应不同 Windows 版本
3. 回退策略：当查询失败时提供合理的默认值

如果您在特定 Windows 版本上遇到问题，请提交 issue 并附上详细的错误日志和 Windows 版本信息。

#### macOS

SysSpector 在 macOS 上使用 `ghw` 包和系统命令收集系统信息。它能够自动检测 Intel 芯片和 M 系列芯片，并使用相应的方法收集信息。

### 依赖

- [github.com/shirou/gopsutil/v3](https://github.com/shirou/gopsutil) - 跨平台硬件监控
- [github.com/StackExchange/wmi](https://github.com/StackExchange/wmi) - Windows WMI 查询

## 项目结构

```
SysSpector/
├── cmd/                  // 主程序入口
│   └── sysinfo/
│       └── main.go       // 初始化和输出逻辑
├── internal/             // 平台特定实现
│   ├── windows/          
│   │   └── wmi.go        // WMI 查询包装
│   └── darwin/
│       └── command.go    // macOS 命令解析
├── pkg/                  // 公共库
│   └── model/
│       └── system.go     // 数据模型定义
├── go.mod
└── README.md             // 项目文档
```

## 许可

MIT
