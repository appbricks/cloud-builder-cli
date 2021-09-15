package space

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder-cli/cmd/target"
	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
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
		ListSpaces()
	},
}

type spaceSelectorArgs struct {
	space      userspace.SpaceNode
	accessType auth.Role
}

var spaceSelector = cbcli_utils.OptionSelector{
	Options: []cbcli_utils.Option{
		{
			Text: " - Connect",
			Command: func(data interface{}) error { 
				space := data.(userspace.SpaceNode)
				if err := run.RunAsAdminWithArgs(
					[]string{
						os.Args[0],
						"space",
						"connect",
						space.GetRecipe(),
						space.GetIaaS(),
						"-r",
						space.GetRegion(),
					},
					os.Stdout, os.Stderr,
				); err != nil {
					logger.DebugMessage(
						"Execution of CLI command with elevated privileges failed with error: %s", 
						err.Error(),
					)
					os.Exit(1)
				} else {
					os.Exit(0)
				}
				return nil
			},
		},
		{
			Text: " - Manage",
			Command: func(data interface{}) error { 
				ManageSpace(data.(userspace.SpaceNode))
				return nil
			},
		},
		{
			Text: " - Suspend",
			Command: func(data interface{}) error { 
				target.SuspendTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
		{
			Text: " - Resume",
			Command: func(data interface{}) error { 
				target.ResumeTarget(data.(userspace.SpaceNode).Key())
				return nil
			},
		},
	},
	OptionListFilter: map[string][]int{
		"undeployed": {},
		"running":    {0, 1, 2},
		"shutdown":   {3},
		"pending":    {},
		"unknown":    {},
	},
	OptionRoleFilter:  map[auth.Role]map[int]bool{
		// owned space administered
		// locally via a device to which
		// the logged in user has admin
		// access
		auth.Admin: {
			0: true, 1: true, 2: true, 3: true,
		},
		// space to which the logged in
		// user has admin access. this
		// could be a shared non-owned
		// space or an owned space on
		// a device to which the logged
		// in user does not have admin
		// access
		auth.Manager: {
			0: true, 1: true,
			/* enable when spaces can be suspended and resumed remotely */
			// 2: true, 3: true,
		},
		// non-owned space shared with
		// the logged user as guest
		auth.Guest: {
			0: true, 
		},		
	},
}

func ListSpaces() {

	var (
		err error

		response string
	)

	spaceIndex := 0
	spaceSubCommandArgs := []spaceSelectorArgs{}

	// shared spaces a retrieved after login for target commands
	spaces := cbcli_config.SpaceNodes.GetAllSpaces()
	if (len(spaces) > 0) {
		fmt.Println("\nYou have access to the following shared spaces.")
		sharedSpacesTable := buildSpacesTable(spaces, &spaceIndex, &spaceSubCommandArgs)
		fmt.Println(color.OpBold.Render("\nMy Cloud Spaces\n================\n"))
		fmt.Println(sharedSpacesTable.Render())
	}

	numSpaces := len(spaceSubCommandArgs)
	if listFlags.actions && numSpaces > 0 {

		optionList := make([]string, spaceIndex)
		for i := 0; i < numSpaces; i++ {
			optionList[i] = strconv.Itoa(i + 1)
		}
		if response = cbcli_utils.GetUserInputFromList(
			"Enter # of node to execute sub-command on or (q)uit: ",
			"", optionList, false); response == "q" {
			fmt.Println()
			return
		}
		if spaceIndex, err = strconv.Atoi(response); err != nil ||
			spaceIndex < 1 || spaceIndex > numSpaces {
			cbcli_utils.ShowErrorAndExit("invalid entry")
		}

		spaceIndex--
		space := spaceSubCommandArgs[spaceIndex].space

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

		if err = spaceSelector.SelectOption(
			space,
			space.GetStatus(),
			spaceSubCommandArgs[spaceIndex].accessType,
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
	}
}

func buildSpacesTable(
	spaces []userspace.SpaceNode,
	spaceIndex *int,
	spaceSubCommandArgs *[]spaceSelectorArgs,
) *termtables.Table  {

	deviceContext := cbcli_config.Config.DeviceContext()

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("Recipe"),
		color.OpBold.Render("Cloud"),
		color.OpBold.Render("Region"),
		color.OpBold.Render("Deployment Name"),
		color.OpBold.Render("Version"),
		color.OpBold.Render("Owned"),
		color.OpBold.Render("Access Type"),
		color.OpBold.Render("Status"),
		color.OpBold.Render("#"),
	)

	for i := 0; i < len(spaces); i++ {

		var (
			spacePrev, space userspace.SpaceNode
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
			spacePrev = nil
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
		owned := ""
		if space.IsSpaceOwned() {
			owned = "yes"
		}
		accessType := auth.RoleFromContext(deviceContext, space)
		
		status = space.GetStatus()
		if (status == "running" || status == "shutdown") {
			*spaceIndex++

			*spaceSubCommandArgs = append(*spaceSubCommandArgs,
				spaceSelectorArgs{
					space:  space,
					accessType: accessType,
				},
			)

			table.AddRow(
				recipe,
				iaas,
				region,
				space.GetSpaceName(),
				space.GetVersion(),
				owned,
				accessType.String(),
				formatStatus(status),
				strconv.Itoa(*spaceIndex),
			)

		} else {
			table.AddRow(
				color.OpFuzzy.Render(recipe),
				color.OpFuzzy.Render(iaas),
				color.OpFuzzy.Render(region),
				color.OpFuzzy.Render(space.GetSpaceName()),
				color.OpFuzzy.Render(space.GetVersion()),
				owned,
				accessType.String(),
				color.OpFuzzy.Render(formatStatus(status)),
				"",
			)
		}
	}

	return table
}

func formatStatus(status string) string {

	var (
		statusName string
	)

	switch status {
	case "undeployed":
		statusName = "not deployed"
	case "running":
		statusName = color.OpReverse.Render(
			color.Green.Render("running"),
		)
	case "shutdown":
		statusName = color.OpReverse.Render(
			color.Red.Render("shutdown"),
		)
	case "pending":
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
