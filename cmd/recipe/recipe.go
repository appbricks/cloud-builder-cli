package recipe

import (
	"github.com/spf13/cobra"
)

var RecipeCommands = &cobra.Command{
	Use: "recipe",

	Short: "List, show detail of recipes and configure recipe defaults.",
	Long: `
The cloud-build CLI include a set of recipes which contain 
instructions on how to launch services in the cloud. The sub-commands
below allow interaction with recipe templates to create customized
targets which can be launched on demand. 
`,
}

func init() {
	RecipeCommands.AddCommand(listCommand)
	RecipeCommands.AddCommand(showCommand)
	RecipeCommands.AddCommand(configureCommand)
}
