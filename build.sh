#!/bin/bash

# 创建输出目录
mkdir -p build

echo "开始编译 SysSpector..."

# 编译 Windows 64位版本
echo "编译 Windows 64位版本..."
GOOS=windows GOARCH=amd64 go build -o build/sysinfo_windows_amd64.exe ./cmd/sysinfo

# 编译 macOS Intel版本
echo "编译 macOS Intel版本..."
GOOS=darwin GOARCH=amd64 go build -o build/sysinfo_macos_intel ./cmd/sysinfo

# 编译 macOS M系列芯片版本
echo "编译 macOS M系列芯片版本..."
GOOS=darwin GOARCH=arm64 go build -o build/sysinfo_macos_arm ./cmd/sysinfo

# 创建Windows打包目录
echo "创建Windows打包目录..."
mkdir -p build/windows
cp build/sysinfo_windows_amd64.exe build/windows/SysSpector.exe

# 复制Windows批处理文件
if [ -f "build/windows/collect_info.bat" ]; then
    echo "Windows批处理文件已存在，跳过复制..."
else
    echo "创建Windows批处理文件..."
    cat > build/windows/collect_info.bat << 'EOF'
@echo off
chcp 65001 > nul
echo 正在收集系统信息...

REM 创建输出目录
mkdir SysInfo

REM 运行SysSpector主程序
echo 运行SysSpector主程序...
SysSpector.exe > SysInfo\sysinfo.json

REM 收集网络信息
echo 收集网络信息...
ipconfig /all > SysInfo\ipconfig.txt
netsh wlan show interfaces > SysInfo\wlan_interfaces.txt
netsh wlan show networks mode=bssid > SysInfo\wlan_networks.txt
netstat -ano > SysInfo\netstat.txt
route print > SysInfo\route.txt

REM 收集系统信息
echo 收集系统信息...
systeminfo > SysInfo\systeminfo.txt
wmic cpu get caption, deviceid, name, numberofcores, maxclockspeed, status /format:list > SysInfo\cpu_info.txt
wmic memorychip get capacity, speed, devicelocator /format:list > SysInfo\memory_info.txt
wmic diskdrive get model, size, status /format:list > SysInfo\disk_info.txt
wmic logicaldisk get caption, description, filesystem, size, freespace /format:list > SysInfo\logical_disk.txt
wmic path win32_VideoController get name, driverversion, currentrefreshrate, videomodedescription /format:list > SysInfo\gpu_info.txt

REM 收集温度信息
echo 收集温度信息...
wmic /namespace:\\root\wmi PATH MSAcpi_ThermalZoneTemperature get CurrentTemperature /format:list > SysInfo\temperature.txt

echo 信息收集完成！所有文件都在 SysInfo 目录中。
EOF
fi

# 创建Windows README文件
if [ -f "build/windows/README.md" ]; then
    echo "Windows README文件已存在，跳过创建..."
else
    echo "创建Windows README文件..."
    cat > build/windows/README.md << 'EOF'
# SysSpector for Windows

## 使用方法

1. 解压缩本压缩包到任意目录
2. 双击运行 `collect_info.bat` 批处理文件
3. 系统将自动收集信息并保存到 `SysInfo` 目录中

## 文件说明

- `SysSpector.exe`: 主程序，用于收集系统硬件信息
- `collect_info.bat`: 批处理文件，用于收集更详细的系统信息
- `SysInfo/sysinfo.json`: 主要系统信息（JSON格式）
- `SysInfo/` 目录下的其他文本文件: 各类详细系统信息

## 注意事项

- 请确保以管理员权限运行批处理文件，以获取完整信息
- 收集的信息仅用于系统诊断，不会上传到互联网
EOF
fi

# 创建Windows打包文件
echo "创建Windows打包文件..."
cd build
zip -r windows/SysSpector_Windows.zip windows/SysSpector.exe windows/collect_info.bat windows/README.md
cd ..

echo "编译完成！所有二进制文件和打包文件都在 build 目录中。"
