package target

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/cloud"

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

	Run: func(cmd *cobra.Command, args []string) {
		ResumeTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(showFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func ResumeTarget(targetKey string) {

	var (
		err error

		tgt *target.Target
	)

	if tgt, err = config.Config.Context().GetTarget(targetKey); err == nil && tgt != nil {

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
						fmt.Printf("Starting instance \"%s\"...", name)
					} else if len(instance.PublicIP()) > 0 {
						for !instance.CanConnect(22) {
							fmt.Print(".")
							time.Sleep(time.Second * 5)
						}
						fmt.Println("done")
					} else {
						fmt.Println("done")
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
