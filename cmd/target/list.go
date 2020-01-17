package target

import (
	"fmt"
	"strconv"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/logger"
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
		color.OpBold.Render("Recipe"),
		color.OpBold.Render("Cloud"),
		color.OpBold.Render("Region"),
		color.OpBold.Render("Deployment Name"),
		color.OpBold.Render("Version"),
		color.OpBold.Render("Status"),
		color.OpBold.Render("#"),
	)

	targetIndex := 1
	targets := config.Config.Context().TargetSet()

	tableRow := make([]interface{}, 7)
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
					tableRow[6] = strconv.Itoa(targetIndex)

					for _, tgt := range targets {
						tableRow[3] = tgt.DeploymentName()

						if err = tgt.LoadRemoteRefs(); err != nil {
							logger.DebugMessage(
								"Error loading target remote references for '%s': %s",
								tgt.Key(), err.Error(),
							)
							tableRow[5] = "error!"

						} else {
							tableRow[5] = getTargetStatusName(tgt)
						}
						tableRow[4] = tgt.Version()

						table.AddRow(tableRow...)
						tableRow[0] = ""
						tableRow[1] = ""
						tableRow[2] = ""

						targetIndex++
					}
					hasTargets = true
				}
			}
			if !hasTargets {
				tableRow[1] = color.OpFuzzy.Render(tableRow[1].(string))
				tableRow[2] = ""
				tableRow[3] = ""
				tableRow[4] = ""
				tableRow[5] = color.OpFuzzy.Render("not configured")
				tableRow[6] = ""
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

func getTargetStatusName(tgt *target.Target) string {

	var (
		statusName string
	)

	switch tgt.Status() {
	case target.Undeployed:
		statusName = "not deployed"
	case target.Running:
		statusName = color.OpReverse.Render(
			color.Green.Render("running"),
		)
	case target.Shutdown:
		statusName = color.OpReverse.Render(
			color.Red.Render("shutdown"),
		)
	case target.Pending:
		statusName = color.OpReverse.Render(
			color.Yellow.Render("pending"),
		)
	default:
		statusName = color.OpFuzzy.Render("unknown")
	}
	return statusName
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&listFlags.stopped, "stopped", "s", false, "list only stopped targets")
	flags.BoolVarP(&listFlags.running, "running", "r", false, "list only running targets")
}
