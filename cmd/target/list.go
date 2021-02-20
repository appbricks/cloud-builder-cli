package target

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/cookbook"
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
	tgt   *target.Target
	state target.TargetState
}

type subCommand struct {
	optionText string
	subCommand func(string)
	setFlags   func()
}

var targetSubCommands = []subCommand{
	{
		optionText: " - Show",
		subCommand: ShowTarget,
		setFlags:   func() {},
	},
	{
		optionText: " - Configure",
		subCommand: ConfigureTarget,
		setFlags:   func() {},
	},
	{
		optionText: " - Launch/Update",
		subCommand: LaunchTarget,
		setFlags:   func() {},
	},
	{
		optionText: " - Delete",
		subCommand: DeleteTarget,
		setFlags: func() {
			deleteFlags.keep = true
		},
	},
	{
		optionText: " - Suspend",
		subCommand: SuspendTarget,
		setFlags:   func() {},
	},
	{
		optionText: " - Resume",
		subCommand: ResumeTarget,
		setFlags:   func() {},
	},
}

var targetOptionsForState = map[target.TargetState][]int{
	target.Undeployed: {0, 1, 2},
	target.Running:    {0, 1, 2, 3, 4},
	target.Shutdown:   {0, 1, 2, 3, 5},
	target.Pending:    {0},
	target.Unknown:    {0},
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

		response string

		targetIndex,
		subCommandIndex int

		spacesRecipes,
		appsRecipes []cookbook.CookbookRecipeInfo
	)
	fmt.Println()
	
	for _, r := range config.Config.Context().Cookbook().RecipeList() {
		if r.IsBastion {
			spacesRecipes = append(spacesRecipes, r)
		} else {
			appsRecipes = append(appsRecipes, r)
		}
	}

	targetIndex = 0
	targetSubCommandArgs := []subCommandArgs{}

	spacesTable := buildSpacesTable(spacesRecipes, &targetIndex, &targetSubCommandArgs)
	appsTable := buildAppsTable(appsRecipes, &targetIndex, &targetSubCommandArgs)

	fmt.Println("\nThe following targets have been configured.")
	fmt.Println(color.OpBold.Render("\nMy Cloud Spaces\n===============\n"))
	if len(spacesRecipes) > 0 {
		fmt.Println(spacesTable.Render())
	} else {
		fmt.Println(color.FgYellow.Render("No space recipes found..."))
	}
	fmt.Println(color.OpBold.Render("My Applications\n===============\n"))
	if len(appsRecipes) > 0 {
		fmt.Println(appsTable.Render())
	} else {
		fmt.Println(color.FgYellow.Render("No application recipes found..."))
	}
	fmt.Println()

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
		tgt := targetSubCommandArgs[targetIndex].tgt

		fmt.Println("\nSelect sub-command to execute on target.")
		fmt.Print("\n  Recipe: ")
		fmt.Println(color.OpBold.Render(tgt.RecipeName))
		fmt.Print("  Cloud:  ")
		fmt.Println(color.OpBold.Render(tgt.RecipeIaas))
		fmt.Print("  Region: ")
		fmt.Println(color.OpBold.Render(*tgt.Provider.Region()))
		fmt.Print("  Name:   ")
		fmt.Println(color.OpBold.Render(tgt.DeploymentName()))
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

			targetSubCommand := targetSubCommands[subCommandIndex-1]
			targetSubCommand.setFlags()
			targetSubCommand.subCommand(tgt.Key())

		} else {
			cbcli_utils.ShowErrorAndExit("invalid option provided")
		}
	}
}

func buildSpacesTable(
	recipes []cookbook.CookbookRecipeInfo,
	targetIndex *int,
	targetSubCommandArgs *[]subCommandArgs,
) *termtables.Table  {

	var (
		err error
		msg string

		hasTargets bool

		lastRecipeIndex,
		lastCloudIndex int
	)

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

						msg = fmt.Sprintf("\rQuerying %s...", tgt.Name())
						fmt.Print(msg)

						if err = tgt.LoadRemoteRefs(); err != nil {
							logger.DebugMessage(
								"Error loading target remote references for '%s': %s",
								tgt.Key(), err.Error(),
							)
							tableRow[5] = "error!"

						} else {
							*targetIndex++

							tableRow[5] = getTargetStatusName(tgt)
							tableRow[6] = strconv.Itoa(*targetIndex)

							*targetSubCommandArgs = append(*targetSubCommandArgs,
								subCommandArgs{
									tgt:   tgt,
									state: tgt.Status(),
								},
							)
						}
						tableRow[3] = tgt.DeploymentName()
						tableRow[4] = tgt.Version()

						table.AddRow(tableRow...)
						tableRow[0] = ""
						tableRow[1] = ""
						tableRow[2] = ""

						fmt.Print("\r")
						utils.RepeatString(" ", len(msg), os.Stdout)
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

	return table
}

func buildAppsTable(
	recipes []cookbook.CookbookRecipeInfo,
	targetIndex *int,
	targetSubCommandArgs *[]subCommandArgs,
) *termtables.Table {

	var (
		err error
		msg string

		hasTargets bool

		lastRecipeIndex,
		lastCloudIndex int
	)

	lastRecipeIndex = len(recipes) - 1
	lastCloudIndex = -1

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("Name"),
		color.OpBold.Render("Cloud"),
		color.OpBold.Render("Attached to Targets"),
		color.OpBold.Render("Deployed App Name"),
		color.OpBold.Render("Version"),
		color.OpBold.Render("Status"),
		color.OpBold.Render("#"),
	)

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
			targets := targets.Lookup(recipe.Name, cloudProvider.Name())

			if len(targets) > 0 {
				for _, tgt := range targets {

					msg = fmt.Sprintf("\rQuerying %s...", tgt.Name())
					fmt.Print(msg)

					if err = tgt.LoadRemoteRefs(); err != nil {
						logger.DebugMessage(
							"Error loading target remote references for '%s': %s",
							tgt.Key(), err.Error(),
						)
						tableRow[5] = "error!"

					} else {
						*targetIndex++

						tableRow[5] = getTargetStatusName(tgt)
						tableRow[6] = strconv.Itoa(*targetIndex)

						*targetSubCommandArgs = append(*targetSubCommandArgs,
							subCommandArgs{
								tgt:   tgt,
								state: tgt.Status(),
							},
						)
					}
					tableRow[3] = tgt.DeploymentName()
					tableRow[4] = tgt.Version()

					// show target dependencies
					for _, dtgt := range tgt.Dependencies() {
						tableRow[2] = dtgt.Key()

						table.AddRow(tableRow...)
						tableRow[4] = ""
						tableRow[5] = ""
						tableRow[6] = ""
					}
					tableRow[0] = ""
					tableRow[1] = ""

					fmt.Print("\r")
					utils.RepeatString(" ", len(msg), os.Stdout)
				}
				hasTargets = true
			}			
			if !hasTargets {
				tableRow[1] = color.OpFuzzy.Render(tableRow[1].(string))
				tableRow[2] = ""
				tableRow[3] = ""
				tableRow[4] = ""
				tableRow[5] = color.OpFuzzy.Render("not configured")
				tableRow[6] = ""
				table.AddRow(tableRow...)
			}

			if j != lastCloudIndex {
				table.AddSeparator()
			}
			tableRow[0] = ""
		}
	}
	return table
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
