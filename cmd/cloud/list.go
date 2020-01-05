package cloud

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/term"
	"github.com/mevansam/termtables"

	"github.com/appbricks/cloud-builder-cli/config"
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
		if listFlags.region {
			ListCloudsByRegion()
		} else {
			ListClouds()
		}
	},
}

func ListClouds() {

	cloudList := config.Config.Context().CloudProviderTemplates()

	table := termtables.CreateTable()
	table.AddHeaders(
		term.BOLD+"Name"+term.NORMAL,
		term.BOLD+"Description"+term.NORMAL,
		term.BOLD+"Configured"+term.NORMAL,
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
				term.DIM+cp.Name()+term.NORMAL,
				term.DIM+cp.Description()+term.NORMAL,
				term.DIM+"no"+term.NORMAL,
			)
		}
	}

	fmt.Printf(
		"\nThis Cloud Builder cookbook supports launching recipes in the public clouds listed below.\n\n%s\n",
		table.Render())
}

func ListCloudsByRegion() {

	cloudList := config.Config.Context().CloudProviderTemplates()

	fmt.Printf("\nThis Cloud Builder cookbook supports launching recipes in the public cloud regions listed below.\n\n")
	for _, cp := range cloudList {

		table := termtables.CreateTable()
		table.AddHeaders(
			term.BOLD+"Region Name"+term.NORMAL,
			term.BOLD+"Description"+term.NORMAL,
		)

		for _, r := range cp.GetRegions() {
			table.AddRow(r.Name, r.Description)
		}

		fmt.Printf(
			term.BOLD+"%s\n%s\n\n%s\n"+term.NORMAL,
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
