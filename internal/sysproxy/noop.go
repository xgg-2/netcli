//go:build !windows

package sysproxy

import (
	"fmt"
	"runtime"
)

func Enable(port int) error {
	fmt.Printf("note: --system-proxy is not yet implemented on %s; configure your proxy manually or use 'netcli run --'\n", runtime.GOOS)
	return nil
}

func Disable() error {
	return nil
}

func DisableDirect() error {
	fmt.Printf("note: restore-proxy is not applicable on %s\n", runtime.GOOS)
	return nil
}

func RegisterCtrlHandler(restore func()) error {
	return nil
}
