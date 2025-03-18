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

echo "编译完成！所有二进制文件都在 build 目录中。"
