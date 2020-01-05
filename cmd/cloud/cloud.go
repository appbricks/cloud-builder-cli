package cloud

import (
	"github.com/spf13/cobra"
)

var CloudCommands = &cobra.Command{
	Use: "cloud [name]",

	Short: "List, show and configure clouds to launch recipes in.",
	Long: `
The cloud sub-commands below allow you to retrieve information 
regarding the available public clouds where you can launch recipes. 
You should also use the "configure" sub-command to set-up your cloud
account access credentials required to create resources in the cloud.
`,
}

func init() {
	CloudCommands.AddCommand(listCommand)
	CloudCommands.AddCommand(showCommand)
	CloudCommands.AddCommand(configureCommand)
}
