package target

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/cloud"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var resumeFlags = struct {
	commonFlags

	instance string
}{}

var resumeCommand = &cobra.Command{
	Use: "resume [recipe] [cloud] [deployment name]",

	Short: "Resumes a suspended target.",
	Long: `
This sub-command resumes all instances deployed to a target. To
resume a specific instance provide the instance name via the 
'-i|--instance' option.
`,

	PreRun: cbcli_auth.AssertAuthorized,

	Run: func(cmd *cobra.Command, args []string) {
		ResumeTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(resumeFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func ResumeTarget(targetKey string) {

	var (
		err error

		tgt *target.Target
		s   *spinner.Spinner
	)

	if tgt, err = cbcli_config.Config.Context().GetTarget(targetKey); err == nil && tgt != nil {

		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if tgt.Status() == target.Shutdown {
			fmt.Println()
			if err = tgt.LoadRemoteRefs(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())

			} else if err = tgt.Resume(
				func(name string, instance cloud.ComputeInstance) {
					state, _ := instance.State()
					if state == cloud.StateStopped {
						s = spinner.New(
							spinner.CharSets[39], 
							100*time.Millisecond,
							spinner.WithSuffix(fmt.Sprintf(" Starting instance \"%s\".", name)),
							spinner.WithFinalMSG(fmt.Sprintf("Instance \"%s\" started.\n\n", name)),
							spinner.WithHiddenCursor(true),
						)
						s.Start()	
						
					} else if len(instance.PublicIP()) > 0 {
						for !instance.CanConnect(22) {
							time.Sleep(time.Second * 5)
						}
						s.Stop()
					} else {
						s.Stop()
					}
				},
			); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		} else {
			cbcli_utils.ShowErrorAndExit("target needs to be 'shutdown' to be resumed")
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
	flags := resumeCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(resumeFlags.commonFlags))

	flags.StringVarP(&resumeFlags.instance, "instance", "i", "", "name of the instance to resume")
}
