//go:build windows

package sysproxy

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	ctrlCEvent        uint32  = 0
	ctrlBreakEvent    uint32  = 1
	ctrlCloseEvent    uint32  = 2
	ctrlLogoffEvent   uint32  = 5
	ctrlShutdownEvent uint32  = 6
	optSettingsChanged uintptr = 39
	optRefresh         uintptr = 37
)

const regPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

var (
	wininet              = windows.NewLazySystemDLL("wininet.dll")
	procInternetSetOption = wininet.NewProc("InternetSetOptionW")

	savedEnable uint64
	savedServer string
)

func broadcast() {
	procInternetSetOption.Call(0, optSettingsChanged, 0, 0)
	procInternetSetOption.Call(0, optRefresh, 0, 0)
}

func Enable(port int) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	savedEnable, _, _ = k.GetIntegerValue("ProxyEnable")
	savedServer, _, _ = k.GetStringValue("ProxyServer")

	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}
	server := fmt.Sprintf("127.0.0.1:%d", port)
	if err := k.SetStringValue("ProxyServer", server); err != nil {
		return fmt.Errorf("set ProxyServer: %w", err)
	}

	broadcast()
	return nil
}

func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", uint32(savedEnable)); err != nil {
		return fmt.Errorf("restore ProxyEnable: %w", err)
	}
	if err := k.SetStringValue("ProxyServer", savedServer); err != nil {
		return fmt.Errorf("restore ProxyServer: %w", err)
	}

	broadcast()
	return nil
}

func DisableDirect() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}

	broadcast()
	return nil
}

func RegisterCtrlHandler(restore func()) error {
	handler := syscall.NewCallback(func(ctrlType uint32) uintptr {
		switch ctrlType {
		case ctrlCEvent, ctrlBreakEvent, ctrlCloseEvent, ctrlLogoffEvent, ctrlShutdownEvent:
			restore()
		}
		return 0
	})
	return windows.SetConsoleCtrlHandler(handler, true)
}
