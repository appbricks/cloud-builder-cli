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
	init         bool
	rebuild      bool
	cleanRebuild bool
	plan         bool
}{}

var launchCommand = &cobra.Command{
	Use: "launch [recipe] [cloud] [region] [deployment name]",

	Short: "Create a launch target.",
	Long: `
Deploys a quick launch target or re-applies any configuration
updates. Rebuild and Clean-rebuild options are complementary and
clean-rebuild takes precedence. 
`,

	Run: func(cmd *cobra.Command, args []string) {
		LaunchTarget(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func LaunchTarget(recipe, cloud, region, deploymentName string) {

	var (
		err error

		tgt  *target.Target
		bldr *target.Builder
	)

	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, cloud, region, deploymentName)
	if tgt, err = config.Config.Context().GetTarget(targetName); err == nil && tgt != nil {

		if bldr, err = tgt.NewBuilder(os.Stdout, os.Stderr); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		fmt.Println()

		if err = tgt.PrepareBackend(); err != nil {
			// ensure backend state storage resources
			// are created
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if launchFlags.init {
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
		} else {
			// deploy target recipe to cloud
			if err = bldr.Launch(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}

			output := bldr.Output()
			logger.TraceMessage("Launch output: %# v", output)

			tgt.Output = output
			config.Config.Context().SaveTarget(tgt.Key(), tgt)
		}
		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown target named \"%s\". Run 'cb target list' "+
					"to list the currently configured targets",
				targetName,
			),
		)
	}
}

func init() {
	flags := launchCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&launchFlags.init, "init", "i", false,
		"re-initialize the launch context")
	flags.BoolVarP(&launchFlags.rebuild, "rebuild", "r", false,
		"re-build/upgrade the target instance resources using the most recent version")
	flags.BoolVarP(&launchFlags.cleanRebuild, "clean-rebuild", "x", false,
		"re-build all instances and attached storage created by the launch recipe")
	flags.BoolVarP(&launchFlags.plan, "plan", "p", false,
		"show cloud resources to be created or changed, but do not launch")
}
