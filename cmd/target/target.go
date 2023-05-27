package target

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/appbricks/cloud-builder/target"
)

var TargetCommands = &cobra.Command{
	Use: "target",

	Short: "List, show detail of running targets and configure quick launch recipes.",
	Long: `
A target is an instance of a recipe that can be launched with a 
single click to a cloud region. When a recipe is configured for a
particular cloud it will enumerate all the regions of that cloud as
quick lauch targets. The sub-commands below allow you to launch, view
amd manage these target spaces.
`,
}

func init() {
	TargetCommands.AddCommand(createCommand)
	TargetCommands.AddCommand(listCommand)
	TargetCommands.AddCommand(showCommand)
	TargetCommands.AddCommand(configureCommand)
	TargetCommands.AddCommand(launchCommand)
	TargetCommands.AddCommand(deleteCommand)
	TargetCommands.AddCommand(suspendCommand)
	TargetCommands.AddCommand(resumeCommand)
	TargetCommands.AddCommand(connectCommand)
	TargetCommands.AddCommand(sshCommand)
}

type commonFlags struct {
	space string
}

func bindCommonFlags(
	flags *pflag.FlagSet, 
	commonFlags *commonFlags,
) {
	flags.StringVarP(&commonFlags.space, "space", "s", "", 
		"application's attached space target name")	
}

func getTargetKeyFromArgs(
	deploymentName string, 
	commonFlags *commonFlags,
) string {

	var (
		targetKey string
	)
	
	if len(commonFlags.space) > 0 {
		targetKey = target.CreateKey(deploymentName, commonFlags.space)
	} else {
		targetKey = target.CreateKey(deploymentName)
	}
	return targetKey
}
