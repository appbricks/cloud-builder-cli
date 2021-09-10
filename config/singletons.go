package config

import (
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/mycloudspace-client/mycscloud"
)

var (
	// Global configuration
	Config config.Config

	// Space Targets
	SpaceNodes *mycscloud.SpaceNodes
)
