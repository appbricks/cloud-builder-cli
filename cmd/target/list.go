package target

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/cloud-builder/target"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"

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

	Run: func(cmd *cobra.Command, args []string) {
		ListTargets()
	},
}

type subCommandArgs struct {
	space      userspace.SpaceNode
	status     string
	accessType auth.Role
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
		optionText: " - Connect",
		subCommand: ConnectTarget,
		setFlags:   func() {},
	},
	{
		optionText: " - Manage",
		subCommand: func(k string) {},
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

var targetOptionsForState = map[string][]int{
	"undeployed": {0, 3, 4},
	"running":    {0, 1, 2, 3, 4, 5, 6},
	"shutdown":   {0, 3, 4, 5, 7},
	"pending":    {0},
	"unknown":    {0},
}

var accessOptionFilter = map[auth.Role]map[int]bool {
	// owned space administered 
	// locally via a device to which
	// the logged in user has admin
	// access
	auth.Admin: {
		0: true, 1: true, 2: true, 3: true,  4: true,  5: true,  6: true,  7: true, 
	},
	// space to which the logged in
	// user has admin access. this
	// could be a shared non-owned
	// space or an owned space on
	// a device to which the logged
	// in user does not have admin
	// access
	auth.Manager: {
		0: true, 1: true, 2: true, 
		/* enable when spaces can be suspended and resumed remotely */
		// 6: true, 7: true,	
	},
	// non-owned space shared with 
	// the logged user as guest
	auth.Guest: {
		0: true, 1: true,
	},
}

var hasOption = func(accessType auth.Role, enabledOptions []int, option int) bool {
	for _, o := range enabledOptions {
		if o == option {
			return accessOptionFilter[accessType][option]
		}
	}
	return false
}

func ListTargets() {

	var (
		err error

		spacesRecipes,
		appsRecipes []cookbook.CookbookRecipeInfo

		response string

		targetIndex,
		subCommandIndex int
	)

	targetIndex = 0
	targetSubCommandArgs := []subCommandArgs{}

	if auth.NewRoleMask(auth.Admin).LoggedInUserHasRole(cbcli_config.Config.DeviceContext(), nil) {
			
		for _, r := range cbcli_config.Config.TargetContext().Cookbook().RecipeList() {
			if r.IsBastion {
				spacesRecipes = append(spacesRecipes, r)
			} else {
				appsRecipes = append(appsRecipes, r)
			}
		}

		spacesTable := buildSpacesTable(spacesRecipes, &targetIndex, &targetSubCommandArgs)
		appsTable := buildAppsTable(appsRecipes, &targetIndex, &targetSubCommandArgs)

		fmt.Println("\nYou have administrative access to the following targets from this device.")
		fmt.Println(color.OpBold.Render("\nMy Cloud Spaces\n===============\n"))
		if len(spacesRecipes) > 0 {
			fmt.Println(spacesTable.Render())
		} else {
			cbcli_utils.ShowInfoMessage("No space recipes found...")
		}
		fmt.Println(color.OpBold.Render("My Applications\n===============\n"))
		if len(appsRecipes) > 0 {
			fmt.Println(appsTable.Render())
		} else {
			cbcli_utils.ShowInfoMessage("No application recipes found...")
		}	
	}

	spaces := cbcli_config.SpaceNodes.GetSharedSpaces()
	if (len(spaces) > 0) {
		fmt.Println("\nYou have access to the following shared spaces.")
		sharedSpacesTable := buildSharedSpacesTable(spaces, &targetIndex, &targetSubCommandArgs)
		fmt.Println(color.OpBold.Render("\nMy Shared Cloud Spaces\n======================\n"))
		fmt.Println(sharedSpacesTable.Render())
	}
	
	numTargets := len(targetSubCommandArgs)
	if listFlags.actions && numTargets > 0 {

		optionList := make([]string, targetIndex)
		for i := 0; i < numTargets; i++ {
			optionList[i] = strconv.Itoa(i + 1)
		}
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # of node to execute sub-command on or (q)uit: ",
			"", optionList); response == "q" {
			fmt.Println()
			return
		}
		if targetIndex, err = strconv.Atoi(response); err != nil ||
			targetIndex < 1 || targetIndex > numTargets {
			cbcli_utils.ShowErrorAndExit("invalid option provided")
		}

		targetIndex--
		space := targetSubCommandArgs[targetIndex].space

		fmt.Println("\nSelect sub-command to execute on target.")
		fmt.Print("\n  Recipe: ")
		fmt.Println(color.OpBold.Render(space.GetRecipe()))
		fmt.Print("  Cloud:  ")
		fmt.Println(color.OpBold.Render(space.GetIaaS()))
		fmt.Print("  Region: ")
		fmt.Println(color.OpBold.Render(space.GetRegion()))
		fmt.Print("  Name:   ")
		fmt.Println(color.OpBold.Render(space.GetSpaceName()))
		fmt.Println()

		subCommandArgs := targetSubCommandArgs[targetIndex]
		enabledOptions := targetOptionsForState[subCommandArgs.status]
		numEnabledOptions := len(enabledOptions)
		for i, c := range targetSubCommands {
			if hasOption(subCommandArgs.accessType, enabledOptions, i) {
				fmt.Print(color.Green.Render(strconv.Itoa(i + 1)))
				fmt.Println(c.optionText)
			} else {
				fmt.Println(color.OpFuzzy.Render(strconv.Itoa(i+1) + c.optionText))
			}
		}
		fmt.Println()

		optionList = make([]string, numEnabledOptions)
		allowedOptions := make(map[int]bool)
		for i, o := range enabledOptions {
			o++
			optionList[i] = strconv.Itoa(o)
			allowedOptions[o] = true
		}
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # of sub-command or (q)uit: ",
			"", optionList); response == "q" {
			fmt.Println()
			return
		}

		if subCommandIndex, err = strconv.Atoi(response); err == nil && allowedOptions[subCommandIndex] {
			targetSubCommand := targetSubCommands[subCommandIndex-1]
			targetSubCommand.setFlags()
			targetSubCommand.subCommand(space.(*target.Target).Key())

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

	targets := cbcli_config.Config.TargetContext().TargetSet()

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
							tableRow[6] = ""

						} else {
							*targetIndex++

							tableRow[5] = getTargetStatusName(tgt)
							tableRow[6] = strconv.Itoa(*targetIndex)

							*targetSubCommandArgs = append(*targetSubCommandArgs,
								subCommandArgs{
									space:      tgt,
									status:     tgt.GetStatus(),
									accessType: auth.Admin,
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

	targets := cbcli_config.Config.TargetContext().TargetSet()

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
						tableRow[6] = ""

					} else {
						*targetIndex++

						tableRow[5] = getTargetStatusName(tgt)
						tableRow[6] = strconv.Itoa(*targetIndex)

						*targetSubCommandArgs = append(*targetSubCommandArgs,
							subCommandArgs{
								space:      tgt,
								status:     tgt.GetStatus(),
								accessType: auth.Admin,
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

func buildSharedSpacesTable(
	spaces []*userspace.Space,
	targetIndex *int,
	targetSubCommandArgs *[]subCommandArgs,
) *termtables.Table  {

	sort.Sort(userspace.SpaceCollection(spaces))

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

	for i := 0; i < len(spaces); i++ {

		var (
			spacePrev, space *userspace.Space
			recipe, iaas, region, status string
		)

		if (i > 0) {
			spacePrev = spaces[i-1]
		}
		space = spaces[i]

		if (spacePrev != nil && space.GetRecipe() == spacePrev.GetRecipe()) {
			recipe = ""
		} else {
			if (spacePrev != nil) {
				table.AddSeparator()
			}
			recipe = space.GetRecipe()
		}
		if (spacePrev != nil && len(iaas) == 0 && space.GetIaaS() == spacePrev.GetIaaS()) {
			iaas = ""
		} else {
			iaas = space.GetIaaS()
		}
		if (spacePrev != nil && len(region) == 0 && space.GetRegion() == spacePrev.GetRegion()) {
			region = ""
		} else {
			region = space.GetRegion()
		}
		status = space.GetStatus()
		if (status == "running") {
			*targetIndex++
			
			accessType := auth.Guest
			if space.HasAdminAccess() {
				accessType = auth.Manager
			}
			*targetSubCommandArgs = append(*targetSubCommandArgs,
				subCommandArgs{
					space:  space,
					status: space.GetStatus(),
					accessType: accessType,
				},
			)

			table.AddRow(
				recipe,
				iaas,
				region,
				space.GetSpaceName(),
				space.GetVersion(),
				status,
				strconv.Itoa(*targetIndex),
			)

			} else {
				table.AddRow(
					color.OpFuzzy.Render(recipe),
					color.OpFuzzy.Render(iaas),
					color.OpFuzzy.Render(region),
					color.OpFuzzy.Render(space.GetSpaceName()),
					color.OpFuzzy.Render(space.GetVersion()),
					color.OpFuzzy.Render(status),
					"",
				)
		}
	}

	return table
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&listFlags.actions, "execute", "e", false, "request execution of an action on a listed target")
}
