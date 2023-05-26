package target

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/mevansam/termtables"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
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

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		ListTargets()
	},
}

var spaceSelector = cbcli_utils.OptionSelector{
	Options: []cbcli_utils.Option{
		{
			Text: " - Show",
			Command: func(data interface{}) error {
				ShowTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
		{
			Text: " - Configure",
			Command: func(data interface{}) error {
				ConfigureTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
		{
			Text: " - Launch/Update",
			Command: func(data interface{}) error {
				LaunchTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
		{
			Text: " - Delete",
			Command: func(data interface{}) error {
				deleteFlags.keep = true
				DeleteTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
		{
			Text: " - Suspend",
			Command: func(data interface{}) error {
				SuspendTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
		{
			Text: " - Resume",
			Command: func(data interface{}) error {
				ResumeTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
	},
	OptionListFilter: map[string][]int{
		"undeployed": {0, 1, 2},
		"running":    {0, 1, 2, 3, 4},
		"shutdown":   {0, 1, 2, 3, 5},
		"pending":    {0},
		"unknown":    {0},
	},
	OptionRoleFilter:  map[auth.Role]map[int]bool{
		// owned space administered
		// locally via a device to which
		// the logged in user has admin
		// access
		auth.Admin: {
			0: true, 1: true, 2: true, 3: true, 4: true, 5: true,
		},
	},
}

func ListTargets() {

	var (
		err error
		ok  bool

		response    string
		targetIndex int

		spacesRecipes,
		appsRecipes []cookbook.CookbookRecipeInfo

		targetKey  string
		targetList []*target.Target
	)
	
	// retrieve recipes grouped into spaces and apps for display
	for _, r := range cbcli_config.Config.TargetContext().Cookbook().RecipeList() {
		if r.IsBastion {
			spacesRecipes = append(spacesRecipes, r)
		} else {
			appsRecipes = append(appsRecipes, r)
		}
	}

	// group targets by recipe, iaas and region
	ts := cbcli_config.Config.TargetContext().TargetSet()
	targets := ts.GetTargets()
	targetMap := make(map[string][]*target.Target)
	for _, tgt := range targets {
		targetKey = createKey(tgt.Recipe.RecipeKey(), tgt.Recipe.RecipeIaaS(), tgt.GetRegion())
		if targetList, ok = targetMap[targetKey]; !ok {
			targetMap[targetKey] = []*target.Target{ tgt }
		} else {
			targetMap[targetKey] = append(targetList, tgt)
		}
	}

	targetIndex = 0
	targetList = []*target.Target{}

	spacesTable := buildSpacesTable(spacesRecipes, targetMap, &targetIndex, &targetList)
	appsTable := buildAppsTable(appsRecipes, targetMap, &targetIndex, &targetList)
	disabledTargetRecipes := ts.GetDisabledTargetRecipes()

	if len(disabledTargetRecipes) > 0 {
		cbcli_utils.ShowWarningMessage(
			"\nThere are configured targets which do not have a " + 
			"corresponding cookbook:recipe in the cookbook repo. " +
			"You need to import the cookbooks below to view and "+
			"manage these missing targets.\n",
		)
		disableTargetRecipeTable := buildDisabledTargetRecipeTable(disabledTargetRecipes)
		fmt.Println(disableTargetRecipeTable.Render())

	} else {
		fmt.Println()
	}

	fmt.Println("The following targets have been configured.")
	fmt.Println(color.OpBold.Render("\nMy Cloud Spaces\n===============\n"))
	if len(spacesRecipes) > 0 {
		fmt.Println(spacesTable.Render())
	} else {
		cbcli_utils.ShowInfoMessage("No space recipes found...\n")
	}
	fmt.Println(color.OpBold.Render("My Applications\n===============\n"))
	if len(appsRecipes) > 0 {
		fmt.Println(appsTable.Render())
	} else {
		cbcli_utils.ShowInfoMessage("No application recipes found...\n")
	}

	numTargets := len(targetList)
	if listFlags.actions && numTargets > 0 {

		options := make([]string, targetIndex)
		for i := 0; i < numTargets; i++ {
			options[i] = strconv.Itoa(i + 1)
		}
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # of node to execute sub-command on or (q)uit: ",
			"", options, false); response == "q" {
			fmt.Println()
			return
		}
		if targetIndex, err = strconv.Atoi(response); err != nil ||
			targetIndex < 1 || targetIndex > numTargets {
			cbcli_utils.ShowErrorAndExit("invalid entry")
		}

		targetIndex--
		tgt := targetList[targetIndex]

		fmt.Println("\nSelect sub-command to execute on target.")
		fmt.Print("\n  Recipe: ")
		fmt.Println(color.OpBold.Render(tgt.Recipe.RecipeKey()))
		fmt.Print("  Cloud:  ")
		fmt.Println(color.OpBold.Render(tgt.Recipe.RecipeIaaS()))
		fmt.Print("  Region: ")
		fmt.Println(color.OpBold.Render(*tgt.Provider.Region()))
		fmt.Print("  Name:   ")
		fmt.Println(color.OpBold.Render(tgt.DeploymentName()))
		fmt.Println()

		if err = spaceSelector.SelectOption(tgt, tgt.GetStatus(), auth.Admin); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
	}
}

func buildSpacesTable(
	recipes []cookbook.CookbookRecipeInfo,
	targetMap map[string][]*target.Target,
	targetIndex *int,
	targetList *[]*target.Target,
) *termtables.Table  {

	var (
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

	tableRow := make([]interface{}, 7)
	for i, recipe := range recipes {
		tableRow[0] = recipe.RecipeKey

		// flag to flag last row of the table which if not flagged
		// will cause double separator lines at the end of the dable
		if i == lastRecipeIndex {
			lastCloudIndex = len(recipe.IaaSList) - 1
		}

		for j, cloudProvider := range recipe.IaaSList {
			tableRow[1] = cloudProvider.Name()

			hasTargets = false
			for _, region := range cloudProvider.GetRegions() {
				targets := targetMap[createKey(recipe.RecipeKey, cloudProvider.Name(), region.Name)]

				if len(targets) > 0 {
					tableRow[2] = region.Name

					for _, tgt := range targets {
						
						hasError := tgt.Error() != nil
						enabled := tgt.CanUpdate() && !hasError

						if hasError {
							tableRow[5] = applyFormat("error!", enabled)
						} else {
							tableRow[5] = applyFormat(getTargetStatusName(tgt), enabled)
						}
						if enabled {
							*targetIndex++
							*targetList = append(*targetList, tgt)
							tableRow[6] = strconv.Itoa(*targetIndex)
						} else {
							tableRow[6] = ""
						}
						tableRow[3] = applyFormat(tgt.DeploymentName(), enabled)
						tableRow[4] = applyFormat(tgt.Version(), enabled)

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

	return table
}

func buildAppsTable(
	recipes []cookbook.CookbookRecipeInfo,
	targetMap map[string][]*target.Target,
	targetIndex *int,
	targetList *[]*target.Target,
) *termtables.Table {

	var (
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
		color.OpBold.Render("Deployed App Name"),
		color.OpBold.Render("Attached to Targets"),
		color.OpBold.Render("Version"),
		color.OpBold.Render("Status"),
		color.OpBold.Render("#"),
	)

	tableRow := make([]interface{}, 8)
	for i, recipe := range recipes {
		tableRow[0] = recipe.RecipeKey

		// flag to flag last row of the table which if not flagged
		// will cause double separator lines at the end of the dable
		if i == lastRecipeIndex {
			lastCloudIndex = len(recipe.IaaSList) - 1
		}

		for j, cloudProvider := range recipe.IaaSList {
			tableRow[1] = cloudProvider.Name()

			hasTargets = false
			for _, region := range cloudProvider.GetRegions() {
				targets := targetMap[createKey(recipe.RecipeKey, cloudProvider.Name(), region.Name)]

				if len(targets) > 0 {
					tableRow[2] = region.Name

					for _, tgt := range targets {

						hasError := tgt.Error() != nil
						enabled := tgt.CanUpdate() && !hasError

						if hasError {
							tableRow[6] = applyFormat("error!", enabled)
						} else {
							tableRow[6] = applyFormat(getTargetStatusName(tgt), enabled)
						}
						if enabled {
							*targetIndex++
							*targetList = append(*targetList, tgt)
							tableRow[7] = strconv.Itoa(*targetIndex)
						} else {
							tableRow[7] = ""
						}
						tableRow[3] = tgt.DeploymentName()
						tableRow[5] = tgt.Version()

						// show target dependencies
						for _, dtgt := range tgt.Dependencies() {
							tableRow[4] = dtgt.Key()

							table.AddRow(tableRow...)
							tableRow[3] = ""
							tableRow[5] = ""
							tableRow[6] = ""
							tableRow[7] = ""
						}
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
				tableRow[5] = ""
				tableRow[6] = color.OpFuzzy.Render("not configured")
				tableRow[7] = ""
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

func applyFormat(text string, enabled bool) string {
	if enabled {
		return text
	} else {
		return color.OpFuzzy.Render(text)
	}
}

func buildDisabledTargetRecipeTable(
	disabledTargetRecipes []cookbook.CookbookRecipeInfo,
) *termtables.Table {

	var (
		lastRecipe *cookbook.CookbookRecipeInfo
	)

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("Cookook"),
		color.OpBold.Render("Version"),
		color.OpBold.Render("Recipe"),
	)

	tableRow := make([]interface{}, 3)
	for _, recipe := range disabledTargetRecipes {
		
		if lastRecipe == nil || lastRecipe.CookbookName != recipe.CookbookName {
			if lastRecipe != nil {
				table.AddSeparator()
			}
			tableRow[0] = recipe.CookbookName
		} else {
			tableRow[0] = ""
		}
		if lastRecipe == nil || lastRecipe.CookbookVersion != recipe.CookbookVersion {
			tableRow[1] = recipe.CookbookVersion
		} else {
			tableRow[1] = ""
		}
		
		tableRow[2] = recipe.RecipeName
		table.AddRow(tableRow...)

		lastRecipe = &recipe
	}

	return table
}

func createKey(keyFields... string) string {
	return strings.Join(keyFields, "/")
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