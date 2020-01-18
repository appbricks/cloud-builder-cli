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

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var listFlags = struct {
	actions bool
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

type subCommandArgs struct {
	recipe,
	iaas,
	region,
	name string

	state target.TargetState
}

type subCommand struct {
	optionText string
	subCommand func(string, string, string, string)
}

var targetSubCommands = []subCommand{
	subCommand{
		optionText: " - Configure",
		subCommand: ConfigureTarget,
	},
	subCommand{
		optionText: " - Launch",
		subCommand: LaunchTarget,
	},
	subCommand{
		optionText: " - Delete",
		subCommand: DeleteTarget,
	},
	subCommand{
		optionText: " - Suspend",
		subCommand: SuspendTarget,
	},
	subCommand{
		optionText: " - Resume",
		subCommand: ResumeTarget,
	},
	subCommand{
		optionText: " - SSH",
		subCommand: SSHTarget,
	},
}

var targetOptionsForState = map[target.TargetState][]int{
	target.Undeployed: []int{0, 1},
	target.Running:    []int{0, 1, 2, 3, 5},
	target.Shutdown:   []int{0, 1, 2, 4},
	target.Pending:    []int{},
	target.Unknown:    []int{},
}

var hasOption = func(enabledOptions []int, option int) bool {
	for _, o := range enabledOptions {
		if o == option {
			return true
		}
	}
	return false
}

func ListTargets() {

	var (
		err error

		hasTargets bool

		lastRecipeIndex,
		lastCloudIndex int

		response string

		targetIndex,
		subCommandIndex int
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

	targetIndex = 0
	targetSubCommandArgs := []subCommandArgs{}

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

					for _, tgt := range targets {
						if err = tgt.LoadRemoteRefs(); err != nil {
							logger.DebugMessage(
								"Error loading target remote references for '%s': %s",
								tgt.Key(), err.Error(),
							)
							tableRow[5] = "error!"

						} else {
							targetIndex++

							tableRow[5] = getTargetStatusName(tgt)
							tableRow[6] = strconv.Itoa(targetIndex)

							targetSubCommandArgs = append(targetSubCommandArgs,
								subCommandArgs{
									recipe: recipe.Name,
									iaas:   cloudProvider.Name(),
									region: region.Name,
									name:   tgt.DeploymentName(),
									state:  tgt.Status(),
								},
							)
						}
						tableRow[3] = tgt.DeploymentName()
						tableRow[4] = tgt.Version()

						table.AddRow(tableRow...)
						tableRow[0] = ""
						tableRow[1] = ""
						tableRow[2] = ""
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

	numTargets := len(targetSubCommandArgs)
	if listFlags.actions && numTargets > 0 {

		options := make([]string, targetIndex)
		for i := 0; i < numTargets; i++ {
			options[i] = strconv.Itoa(i + 1)
		}
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # of node to execute sub-command on or (q)uit: ",
			"", options); response == "q" {
			return
		}
		if targetIndex, err = strconv.Atoi(response); err != nil ||
			targetIndex < 1 || targetIndex > numTargets {
			cbcli_utils.ShowErrorAndExit("invalid option provided")
		}

		targetIndex--
		recipe := targetSubCommandArgs[targetIndex].recipe
		iaas := targetSubCommandArgs[targetIndex].iaas
		region := targetSubCommandArgs[targetIndex].region
		name := targetSubCommandArgs[targetIndex].name

		fmt.Println("\nSelect sub-command to execute on target.")
		fmt.Print("\n  Recipe: ")
		fmt.Println(color.OpBold.Render(recipe))
		fmt.Print("  Cloud:  ")
		fmt.Println(color.OpBold.Render(iaas))
		fmt.Print("  Region: ")
		fmt.Println(color.OpBold.Render(region))
		fmt.Print("  Name:   ")
		fmt.Println(color.OpBold.Render(name))
		fmt.Println()

		enabledOptions := targetOptionsForState[targetSubCommandArgs[targetIndex].state]
		numEnabledOptions := len(enabledOptions)
		for i, c := range targetSubCommands {
			if hasOption(enabledOptions, i) {
				fmt.Print(color.Green.Render(strconv.Itoa(i + 1)))
				fmt.Println(c.optionText)
			} else {
				fmt.Println(color.OpFuzzy.Render(strconv.Itoa(i+1) + c.optionText))
			}
		}
		fmt.Println()

		options = make([]string, numEnabledOptions)
		for i, o := range enabledOptions {
			options[i] = strconv.Itoa(o + 1)
		}
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # of sub-command or (q)uit: ",
			"", options); response == "q" {
			return
		}

		if subCommandIndex, err = strconv.Atoi(response); err == nil &&
			hasOption(enabledOptions, subCommandIndex-1) {
			targetSubCommands[subCommandIndex-1].subCommand(recipe, iaas, region, name)
		} else {
			cbcli_utils.ShowErrorAndExit("invalid option provided")
		}
	}
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
	flags.BoolVarP(&listFlags.actions, "execute", "e", false, "request execution of an action on a listed target")
}
