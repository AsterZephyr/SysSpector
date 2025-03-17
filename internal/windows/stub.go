//go:build !windows
// +build !windows

package windows

import (
	"errors"

	"github.com/AsterZephyr/SysSpector/pkg/model"
)

// GetSystemInfo returns an error on non-Windows platforms
func GetSystemInfo() (model.SystemInfo, error) {
	return model.SystemInfo{}, errors.New("windows implementation is not available on this platform")
}
