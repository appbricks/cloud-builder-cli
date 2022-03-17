// +build windows

package space

import (
	"syscall"

	"github.com/mevansam/goutils/logger"
)

const (
	CTRL_C_EVENT        = uint32(0)
	CTRL_BREAK_EVENT    = uint32(1)
	CTRL_CLOSE_EVENT    = uint32(2)
	CTRL_LOGOFF_EVENT   = uint32(5)
	CTRL_SHUTDOWN_EVENT = uint32(6)
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleCtrlHandler = kernel32.NewProc("SetConsoleCtrlHandler")
)

func __overrideInterruptEvent(disconnect chan bool) error {
	
	ok, _, err := procSetConsoleCtrlHandler.Call(
		syscall.NewCallback(func(controlType uint32) uint {
			if controlType == CTRL_C_EVENT ||
				controlType == CTRL_BREAK_EVENT ||
				controlType == CTRL_CLOSE_EVENT ||
				controlType == CTRL_LOGOFF_EVENT ||
				controlType == CTRL_SHUTDOWN_EVENT {

				logger.DebugMessage(
					"__overrideInterruptEvent: Received interrupt event: %d", 
					controlType,
				)
				// ensure all listeners 
				// receive the event
				disconnect <- true
				disconnect <- true

				return 1
			}
			return 0
		}),
		1,
	)
	if ok == 0 {
		return err
	}
	return nil
}

func init() {
	overrideInterruptEvent = __overrideInterruptEvent
}
