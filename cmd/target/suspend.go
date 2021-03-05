package target

import (
	"fmt"

	"github.com/mevansam/gocloud/cloud"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var suspendFlags = struct {
	commonFlags

	instance string
}{}

var suspendCommand = &cobra.Command{
	Use: "suspend [recipe] [cloud] [deployment name]",

	Short: "Suspends a running target.",
	Long: `
This sub-command suspends all instances deployed to a target. To
suspend a specific instance provide the instance name via the 
'-i|--instance' option.
`,

	Run: func(cmd *cobra.Command, args []string) {
		SuspendTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(showFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func SuspendTarget(targetKey string) {

	var (
		err error

		tgt *target.Target
	)

	if tgt, err = config.Config.Context().GetTarget(targetKey); err == nil && tgt != nil {

		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if tgt.Status() == target.Running {
			fmt.Println()
			if err = tgt.LoadRemoteRefs(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())

			} else if err = tgt.Suspend(
				func(name string, instance cloud.ComputeInstance) {
					state, _ := instance.State()
					if state == cloud.StateRunning {
						fmt.Printf("Stopping instance \"%s\"...", name)
					} else {
						fmt.Println("done")
					}
				},
			); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		} else {
			cbcli_utils.ShowErrorAndExit("target needs to be 'running' to be suspended")
		}
		return
	}

	cbcli_utils.ShowErrorAndExit(
		fmt.Sprintf(
			"Target \"%s\" does not exist. Run 'cb target list' to list the currently configured targets",
			targetKey,
		),
	)
}

func init() {
	flags := suspendCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(suspendFlags.commonFlags))

	flags.StringVarP(&suspendFlags.instance, "instance", "i", "", "name of the instance to suspend")
}
