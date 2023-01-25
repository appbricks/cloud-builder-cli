package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/mycloudspace-client/system"
)

var versionCommand = &cobra.Command{
	Use: "version",

	Short: "Show the version of the Cloud Builder CLI.",
	Long: `
Display the version of the Cloud Builder CLI and any other relevant
configuration status.
`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("\nDevice Type:     %s\n", system.GetDeviceType())
		fmt.Printf("Client Version:  %s\n", system.GetDeviceVersion(config.ClientType, config.Version))
		fmt.Printf("Build date:      %s\n\n", config.BuildTimestamp)
	},
}
