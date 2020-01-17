package target

import (
	"github.com/spf13/cobra"
)

var TargetCommands = &cobra.Command{
	Use: "target",

	Short: "List, show detail of running targets and configure quick launch recipes.",
	Long: `
A target is an instance of a recipe that can be launched with a 
single click to a cloud region. When a recipe is configured for a
particular cloud it will enumerate all the regions of that cloud as
quick lauch targets. The sub-commands below allow you to launch and
view the status targets.
`,
}

func init() {
	TargetCommands.AddCommand(listCommand)
	TargetCommands.AddCommand(showCommand)
	TargetCommands.AddCommand(createCommand)
	TargetCommands.AddCommand(configureCommand)
	TargetCommands.AddCommand(launchCommand)
	TargetCommands.AddCommand(deleteCommand)
	TargetCommands.AddCommand(suspendCommand)
	TargetCommands.AddCommand(resumeCommand)
	TargetCommands.AddCommand(sshCommand)
}
