package config

import (
	"runtime"

	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/appbricks/mycloudspace-common/monitors"
	"github.com/briandowns/spinner"
)

var (
	// Global configuration
	Config config.Config

	// Monitor Service
	MonitorService *monitors.MonitorService

	// Space Targets
	SpaceNodes *mycscloud.SpaceNodes

	// Shutdown spinner
	ShutdownSpinner *spinner.Spinner

	SpinnerShutdownType, 
	SpinnerNetworkType int
)

func init() {

	// pick spinner charset that will be correctly 
	// displayed on the os the cli is running on
	SpinnerShutdownType = 11
	SpinnerNetworkType = 39
	if runtime.GOOS == "windows" {
		SpinnerShutdownType = 9
		SpinnerNetworkType = 9
	}
}
