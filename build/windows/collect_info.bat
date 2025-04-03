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

REM 收集进程信息
echo 收集进程信息...
tasklist /v > SysInfo\processes.txt
wmic process get caption, commandline, processid, parentprocessid /format:list > SysInfo\process_details.txt

echo 系统信息收集完成！所有结果保存在 SysInfo 目录中。
echo 按任意键退出...
pause > nul
