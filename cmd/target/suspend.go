package target

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/mevansam/gocloud/cloud"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
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

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		SuspendTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(suspendFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func SuspendTarget(targetKey string) {

	var (
		err error

		tgt *target.Target		
		s   *spinner.Spinner
	)

	if tgt, err = cbcli_config.Config.TargetContext().GetTarget(targetKey); err == nil && tgt != nil {

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
						s = spinner.New(
							spinner.CharSets[39], 
							100*time.Millisecond,
							spinner.WithSuffix(fmt.Sprintf(" Stopping instance \"%s\".", name)),
							spinner.WithFinalMSG(fmt.Sprintf("Instance \"%s\" stopped.\n", name)),
							spinner.WithHiddenCursor(true),
						)
						s.Start()						
					} else {
						s.Stop()
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
