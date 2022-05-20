package device

import (
	"github.com/spf13/cobra"
)

var DeviceCommands = &cobra.Command{
	Use: "device [name]",

	Short: "List, add and delete devices used to access you private cloud space.",
	Long: `
The device sub-commands allow you to list, add and delete devices
that you can use to connect, access and manage you private cloud
space resources.
`,
}

func init() {
	DeviceCommands.AddCommand(listCommand)
	DeviceCommands.AddCommand(addCommand)
	DeviceCommands.AddCommand(deleteCommand)
	DeviceCommands.AddCommand(addUserCommand)
	DeviceCommands.AddCommand(deleteUserCommand)
}
