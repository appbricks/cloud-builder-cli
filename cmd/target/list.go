package target

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/term"
	"github.com/mevansam/termtables"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"
)

var listFlags = struct {
	stopped bool
	running bool
}{}

var listCommand = &cobra.Command{
	Use: "list",

	Short: "List configured targets.",
	Long: `
List all available quick launch targets which are configured recipe
instances. Targets will be enumerated only for clouds a recipe has
been configured for.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ListTargets()
	},
}

func ListTargets() {

	var (
		err error

		hasTargets bool
		// target     *target.Target

		lastRecipeIndex,
		lastCloudIndex int
	)

	recipes := config.Config.Context().Cookbook().RecipeList()
	lastRecipeIndex = len(recipes) - 1
	lastCloudIndex = -1

	table := termtables.CreateTable()
	table.AddHeaders(
		term.BOLD+"Recipe"+term.NORMAL,
		term.BOLD+"Cloud"+term.NORMAL,
		term.BOLD+"Region"+term.NORMAL,
		term.BOLD+"Deployment Name"+term.NORMAL,
		term.BOLD+"Status"+term.NORMAL,
	)

	targets := config.Config.Context().TargetSet()

	tableRow := make([]interface{}, 5)
	for i, recipe := range recipes {
		tableRow[0] = recipe.Name

		// flag to flag last row of the table which if not flagged
		// will cause double separator lines at the end of the dable
		if i == lastRecipeIndex {
			lastCloudIndex = len(recipe.IaaSList) - 1
		}

		for j, cloudProvider := range recipe.IaaSList {
			tableRow[1] = cloudProvider.Name()

			hasTargets = false
			for _, region := range cloudProvider.GetRegions() {
				targets := targets.Lookup(recipe.Name, cloudProvider.Name(), region.Name)

				if len(targets) > 0 {
					tableRow[2] = region.Name

					for _, tgt := range targets {
						tableRow[3] = tgt.DeploymentName()
						if err = tgt.LoadRemoteRefs(); err != nil {
							logger.DebugMessage(
								"Error loading target remote references for '%s': %s",
								tgt.Key(), err.Error(),
							)
							tableRow[4] = "error!"

						} else {
							switch tgt.Status() {
							case target.Undeployed:
								tableRow[4] = "not deployed"
							case target.Running:
								tableRow[4] = "running"
							case target.Shutdown:
								tableRow[4] = "shutdown"
							case target.Pending:
								tableRow[4] = "pending"
							case target.Unknown:
								tableRow[4] = "unknown"
							}
						}

						table.AddRow(tableRow...)
						tableRow[0] = ""
						tableRow[1] = ""
						tableRow[2] = ""
					}
					hasTargets = true
				}
			}
			if !hasTargets {
				tableRow[1] = term.DIM + tableRow[1].(string) + term.NORMAL
				tableRow[2] = ""
				tableRow[3] = ""
				tableRow[4] = term.DIM + "not configured" + term.NORMAL
				table.AddRow(tableRow...)

				tableRow[0] = ""
			}

			if j != lastCloudIndex {
				table.AddSeparator()
			}
		}
	}

	fmt.Printf(
		"\nThe following recipe targets have been configured.\n\n%s\n",
		table.Render())
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&listFlags.stopped, "stopped", "s", false, "list only stopped targets")
	flags.BoolVarP(&listFlags.running, "running", "r", false, "list only running targets")
}
