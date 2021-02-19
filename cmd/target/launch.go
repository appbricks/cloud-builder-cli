package target

import (
	"fmt"
	"os"

	"github.com/mevansam/goutils/logger"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var launchFlags = struct {
	commonFlags

	init         bool
	rebuild      bool
	cleanRebuild bool
	plan         bool
}{}

var launchCommand = &cobra.Command{
	Use: "launch [recipe] [cloud] [deployment name]",

	Short: "Deploy a launch target to the cloud.",
	Long: `
Deploys a quick launch target or re-applies any configuration
updates. Rebuild and Clean-rebuild options are complementary and
clean-rebuild takes precedence. 
`,

	Run: func(cmd *cobra.Command, args []string) {
		LaunchTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(showFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func LaunchTarget(targetKey string) {

	var (
		err error

		tgt, spaceTgt *target.Target
		bldr          *target.Builder
	)
	context := config.Config.Context()

	if tgt, err = context.GetTarget(targetKey); err == nil && tgt != nil {

		// ensure any dependencies have been deployed
		if len(showFlags.commonFlags.space) > 0 {
			if spaceTgt, err = context.GetTarget(showFlags.commonFlags.space); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			if spaceTgt.Status() == target.Undeployed {
				cbcli_utils.ShowErrorAndExit("Space target to launch application in has not been deployed.")
			}
		}

		if bldr, err = tgt.NewBuilder(os.Stdout, os.Stderr); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		fmt.Println()

		if err = tgt.PrepareBackend(); err != nil {
			// ensure backend state storage resources
			// are created
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if launchFlags.init ||
			tgt.CookbookTimestamp != tgt.Recipe.CookbookTimestamp() {
			// force re-initializing
			if err = bldr.Initialize(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		} else {
			// initialize if not initialized
			if err = bldr.AutoInitialize(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}

		if launchFlags.cleanRebuild {
			// mark target instance resource data to be
			// rebuilt on next launch
			if err = bldr.SetRebuildInstanceData(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}
		if launchFlags.cleanRebuild || launchFlags.rebuild {
			// mark target instance resources to be
			// rebuilt on next launch
			if err = bldr.SetRebuildInstances(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}

		if launchFlags.plan {
			// show launch plan
			if err = bldr.ShowLaunchPlan(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			tgt.CookbookTimestamp = tgt.Recipe.CookbookTimestamp()
			context.SaveTarget(tgt.Key(), tgt)

		} else {
			// deploy target recipe to cloud
			if err = bldr.Launch(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}

			// retrieve the output of the deployment
			output := bldr.Output()
			logger.TraceMessage("Launch output: %# v", output)

			tgt.Output = output
			tgt.CookbookTimestamp = tgt.Recipe.CookbookTimestamp()
			context.SaveTarget(tgt.Key(), tgt)

			if err = tgt.LoadRemoteRefs(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showNodeInfo(tgt)
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
	flags := launchCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(showFlags.commonFlags))

	flags.BoolVarP(&launchFlags.init, "init", "i", false,
		"re-initialize the launch context")
	flags.BoolVarP(&launchFlags.rebuild, "rebuild", "b", false,
		"re-build/upgrade the target instance resources using the most recent version")
	flags.BoolVarP(&launchFlags.cleanRebuild, "clean-rebuild", "x", false,
		"re-build all instances and attached storage created by the launch recipe")
	flags.BoolVarP(&launchFlags.plan, "plan", "p", false,
		"show cloud resources to be created or changed, but do not launch")
}
