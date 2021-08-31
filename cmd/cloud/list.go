package cloud

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/termtables"

	"github.com/appbricks/cloud-builder/auth"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var listFlags = struct {
	region bool
}{}

var listCommand = &cobra.Command{
	Use: "list",

	Short: "List the public clouds where recipes can be launched.",
	Long: `
Show a list of public clouds and region information where recipes can
be launched. To be able to target a recipe to one of these clouds you 
need ensure your public account credentials have been configured with 
the correct permissions.
`,

	Run: func(cmd *cobra.Command, args []string) {
		cbcli_utils.AssertAuthorized(cmd,
			auth.NewRoleMask(auth.Admin).LoggedInUserHasRole(cbcli_config.Config.DeviceContext()))

		if listFlags.region {
			ListCloudsByRegion()
		} else {
			ListClouds()
		}
	},
}

func ListClouds() {

	cloudList := cbcli_config.Config.Context().CloudProviderTemplates()

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("Name"),
		color.OpBold.Render("Description"),
		color.OpBold.Render("Configured"),
	)

	for _, cp := range cloudList {

		if cp.IsValid() {
			table.AddRow(
				cp.Name(),
				cp.Description(),
				"yes",
			)
		} else {
			table.AddRow(
				color.OpFuzzy.Render(cp.Name()),
				color.OpFuzzy.Render(cp.Description()),
				color.OpFuzzy.Render("no"),
			)
		}
	}

	fmt.Printf(
		"\nThis Cloud Builder cookbook supports launching recipes in the public clouds listed below.\n\n%s\n",
		table.Render())
}

func ListCloudsByRegion() {

	cloudList := cbcli_config.Config.Context().CloudProviderTemplates()

	fmt.Printf("\nThis Cloud Builder cookbook supports launching recipes in the public cloud regions listed below.\n\n")
	for _, cp := range cloudList {

		table := termtables.CreateTable()
		table.AddHeaders(
			color.OpBold.Render("Region Name"),
			color.OpBold.Render("Description"),
		)

		for _, r := range cp.GetRegions() {
			table.AddRow(r.Name, r.Description)
		}

		fmt.Printf(
			color.OpBold.Render("%s\n%s\n\n%s\n"),
			cp.Description(),
			strings.Repeat("=", len(cp.Description())),
			table.Render())
	}
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false
	flags.BoolVarP(&listFlags.region, "regions", "r", false, "show cloud regions")
}
