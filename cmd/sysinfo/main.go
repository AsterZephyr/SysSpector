package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/AsterZephyr/SysSpector/internal/darwin"
	"github.com/AsterZephyr/SysSpector/internal/windows"
	"github.com/AsterZephyr/SysSpector/pkg/model"
)

func main() {
	log.Println("Starting system information collection...")

	var sysInfo model.SystemInfo
	var err error

	// Determine OS and collect information accordingly
	switch runtime.GOOS {
	case "windows":
		log.Println("Detected Windows OS")
		sysInfo, err = windows.GetSystemInfo()
	case "darwin":
		log.Println("Detected macOS")
		sysInfo, err = darwin.GetSystemInfo()
	default:
		log.Fatalf("Unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		log.Printf("Warning: Some information could not be collected: %v", err)
	}

	// Print system information in JSON format
	jsonData, err := json.MarshalIndent(sysInfo, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling system info to JSON: %v", err)
	}

	fmt.Println(string(jsonData))

	// Optionally save to file
	if len(os.Args) > 1 && os.Args[1] == "--save" {
		outputFile := "sysinfo.json"
		if len(os.Args) > 2 {
			outputFile = os.Args[2]
		}

		err = os.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			log.Fatalf("Error writing to file %s: %v", outputFile, err)
		}
		log.Printf("System information saved to %s", outputFile)
	}
}
