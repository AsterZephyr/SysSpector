#!/bin/bash

# 电源和电池信息
collect_battery_info() {
    echo "电池信息:"
    pmset -g batt | tail -n 1 | while read -r line; do
        # 提取电量百分比
        charge_percent=$(echo "$line" | grep -oE '[0-9]+%' | sed 's/%//')
        
        # 判断充电状态
        charging_status=$(echo "$line" | grep -q "charging" && echo "是" || echo "否")
        
        # 低电量判断
        low_battery=$([ "$charge_percent" -lt 20 ] && echo "是" || echo "否")
        
        echo "电量信息: $charge_percent%"
        echo "正在充电: $charging_status"
        echo "电池电量低于警告水平: $low_battery"
    done
}

# 电池健康信息
collect_battery_health() {
    echo -e "\n电池健康:"
    system_profiler SPPowerDataType | awk '
        /Cycle Count:/ {cycle = $3}
        /Condition:/ {condition = $2}
        /Maximum Capacity:/ {capacity = $3}
        END {
            print "循环计数: " cycle
            print "电池状态: " condition
            print "最大容量: " capacity
        }
    '
}

# 充电器信息
collect_charger_info() {
    echo -e "\n充电器信息:"
    # 使用精确的 grep 和 sed 提取充电器信息
    serial=$(system_profiler SPPowerDataType | grep -A 10 "AC Charger Information:" | grep "Serial Number:" | sed 's/.*Serial Number: *//')
    name=$(system_profiler SPPowerDataType | grep -A 10 "AC Charger Information:" | grep "Name:" | sed 's/.*Name: *//')
    
    echo "交流充电器信息-序列号: ${serial:-未找到}"
    echo "交流充电器信息-名称: ${name:-未找到}"
}

# 蓝牙状态
collect_bluetooth_status() {
    echo -e "\n蓝牙信息:"
    bluetooth_status=$(system_profiler SPBluetoothDataType | grep "State:" | awk '{print $2}')
    echo "蓝牙-状态: $bluetooth_status"
}

# 蓝牙连接设备
collect_bluetooth_devices() {
    echo "蓝牙-连接设备:"
    # 使用awk提取'Connected:'下的设备名称
    devices=$(system_profiler SPBluetoothDataType | awk '/Connected:/,/^$/{if($0 ~ /^[[:space:]]*[^:]+:[[:space:]]*$/) {sub(/:$/, "", $0); print $0}}' | sed 's/^[[:space:]]*//g' | tr '\n' '、' | sed 's/、$//')

    if [ -z "$devices" ]; then
        devices="未找到已连接设备"
    fi
    
    echo "$devices"
}
    
# 主函数
main() {
    collect_battery_info
    collect_battery_health
    collect_charger_info
    collect_bluetooth_status
    collect_bluetooth_devices
}

# 执行主函数
main
