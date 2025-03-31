//go:build darwin

// Package smc 提供了访问macOS系统管理控制器(SMC)的功能
// 用于获取CPU和GPU温度等硬件信息
package smc

/*
#cgo LDFLAGS: -framework IOKit
#include <IOKit/IOKitLib.h>
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <mach/mach.h>

typedef struct {
    char key[5];
    uint32_t dataSize;
    uint32_t dataType;
    uint8_t bytes[32];
} SMCVal_t;

static io_connect_t connection;

int SMCOpen() {
    kern_return_t result;
    mach_port_t masterPort;
    io_iterator_t iterator;
    io_object_t device;

    result = IOMasterPort(MACH_PORT_NULL, &masterPort);
    if (result != KERN_SUCCESS) {
        return -1;
    }

    CFMutableDictionaryRef matchingDictionary = IOServiceMatching("AppleSMC");
    if (matchingDictionary == NULL) {
        return -2;
    }

    result = IOServiceGetMatchingServices(masterPort, matchingDictionary, &iterator);
    if (result != KERN_SUCCESS) {
        return -3;
    }

    device = IOIteratorNext(iterator);
    IOObjectRelease(iterator);
    if (device == 0) {
        return -4;
    }

    result = IOServiceOpen(device, mach_task_self(), 0, &connection);
    IOObjectRelease(device);
    if (result != KERN_SUCCESS) {
        return -5;
    }

    return 0;
}

void SMCClose() {
    if (connection != 0) {
        IOServiceClose(connection);
        connection = 0;
    }
}

// 将四字符键转换为uint32_t
uint32_t _strtoul(const char *str, int size, int base) {
    uint32_t total = 0;
    int i;

    for (i = 0; i < size; i++) {
        if (base == 16) {
            total += str[i] << (size - 1 - i) * 8;
        } else {
            total += (unsigned char)(str[i] << (size - 1 - i) * 8);
        }
    }
    return total;
}

// 从SMC读取键值
kern_return_t SMCReadKey(uint32_t key, SMCVal_t *val) {
    kern_return_t result;
    uint32_t keyValue = key;
    size_t outputSize = sizeof(SMCVal_t);
    
    val->dataSize = 0;

    result = IOConnectCallStructMethod(connection, 2, &keyValue, sizeof(keyValue), val, &outputSize);
    if (result != KERN_SUCCESS) {
        return result;
    }

    result = IOConnectCallStructMethod(connection, 5, &keyValue, sizeof(keyValue), val, &outputSize);
    if (result != KERN_SUCCESS) {
        return result;
    }

    return KERN_SUCCESS;
}

// 获取CPU温度
float getCPUTemperature(const char *key) {
    SMCVal_t val;
    kern_return_t result;
    
    uint32_t intKey = _strtoul(key, strlen(key), 16);
    result = SMCReadKey(intKey, &val);
    if (result != KERN_SUCCESS) {
        printf("Failed to read key %s, error: %d\n", key, result);
        return -1.0;
    }

    // 打印调试信息
    printf("Key: %s, DataSize: %d, DataType: 0x%08x\n", key, val.dataSize, val.dataType);
    printf("Bytes: %02x %02x %02x %02x\n", val.bytes[0], val.bytes[1], val.bytes[2], val.bytes[3]);

    // 温度通常存储为SP78类型（定点数，8位整数部分，8位小数部分）
    if (val.dataSize > 0) {
        // 支持多种数据类型
        if (val.dataType == 0x73703738 || val.dataType == 0x53503738) { // "sp78" or "SP78"
            // SP78格式：8位整数部分，8位小数部分
            int intValue = val.bytes[0];
            float fracValue = val.bytes[1] / 256.0;
            float temp = intValue + fracValue;
            printf("SP78 format, temp: %.2f\n", temp);
            return temp;
        } else if (val.dataType == 0x66703238) { // "fp2e"
            // FP2E格式：16位整数部分，2位小数部分
            int intValue = (val.bytes[0] << 8) + val.bytes[1];
            float temp = intValue / 4.0;
            printf("FP2E format, temp: %.2f\n", temp);
            return temp;
        } else if (val.dataType == 0x666c7438) { // "flt "
            // 浮点数格式
            float temp;
            memcpy(&temp, val.bytes, sizeof(float));
            printf("FLT format, temp: %.2f\n", temp);
            return temp;
        } else {
            // 其他格式，尝试按照通用方式解析
            float temp = ((val.bytes[0] * 256 + val.bytes[1]) >> 2) / 64.0;
            printf("Unknown format, temp: %.2f\n", temp);
            return temp;
        }
    }
    
    return -1.0;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// 可能的温度键列表
var possibleTempKeys = []string{
	// Apple Silicon CPU温度键
	"Tp01", "Tp09", "Tp0D", "Tp0P", "Tp0T", "Tp0b", "Tp0f", "Tp0t",
	// Intel CPU温度键
	"TC0D", "TC0P", "TC0E", "TC0F", "TCSA",
	// 通用CPU温度键
	"TCGC", "TCGc",
}

// 可能的GPU温度键列表
var possibleGPUTempKeys = []string{
	// Apple Silicon GPU温度键
	"Tg0D", "Tg0P", "Tg0T",
	// Intel GPU温度键
	"TG0D", "TG0P", "TG0T",
}

// TemperatureInfo 存储温度信息
type TemperatureInfo struct {
	CPUTemp float64
	GPUTemp float64
	KeyUsed string
}

// GetTemperature 获取系统温度信息
func GetTemperature() (TemperatureInfo, error) {
	var info TemperatureInfo
	info.CPUTemp = -1
	info.GPUTemp = -1

	// 打开SMC连接
	if C.SMCOpen() != 0 {
		return info, fmt.Errorf("无法打开SMC连接")
	}
	defer C.SMCClose()

	// 尝试获取CPU温度
	for _, key := range possibleTempKeys {
		cKey := C.CString(key)
		defer C.free(unsafe.Pointer(cKey))

		temp := float64(C.getCPUTemperature(cKey))
		if temp > 0 && temp < 120 { // 添加合理性检查，温度应该在0-120°C之间
			info.CPUTemp = temp
			info.KeyUsed = key
			break
		}
	}

	// 尝试获取GPU温度
	for _, key := range possibleGPUTempKeys {
		cKey := C.CString(key)
		defer C.free(unsafe.Pointer(cKey))

		temp := float64(C.getCPUTemperature(cKey))
		if temp > 0 && temp < 120 { // 添加合理性检查，温度应该在0-120°C之间
			info.GPUTemp = temp
			break
		}
	}

	// 如果没有找到CPU温度，返回错误
	if info.CPUTemp < 0 {
		return info, fmt.Errorf("无法获取CPU温度")
	}

	return info, nil
}
