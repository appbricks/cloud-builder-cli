package cloud

import (
	"github.com/spf13/cobra"
)

var CloudCommands = &cobra.Command{
	Use: "cloud [name]",

	Short: "List, show and configure clouds to launch recipes in.",
	Long: `
The cloud-builder CLI allows you to launch applications and services
in the public cloud. The sub-commands below below allow you to 
view information regarding the available public clouds as well as
configure access to them in order to launch cloud recipes.
`,
}

func init() {
	CloudCommands.AddCommand(listCommand)
	CloudCommands.AddCommand(showCommand)
	CloudCommands.AddCommand(configureCommand)
}
