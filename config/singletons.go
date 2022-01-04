package config

import (
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
)
