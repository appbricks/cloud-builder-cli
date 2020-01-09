package recipe

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"

	"github.com/appbricks/cloud-builder-cli/config"
)

var listFlags = struct {
	cloud string
}{}

var listCommand = &cobra.Command{
	Use: "list",

	Short: "List recipes that can be launch in the cloud.",
	Long: `
Lists the recipes bundled with the CLI that can be launched in any
one of the supported public clouds.
`,

	Run: func(cmd *cobra.Command, args []string) {
		if len(listFlags.cloud) > 0 {
			ListRecipesForCloud(listFlags.cloud)
		} else {
			ListRecipes()
		}
	},
}

func ListRecipes() {

	var (
		clouds, descLines []string

		numLines int
	)

	recipes := config.Config.Context().Cookbook().RecipeList()
	lastRecipesIndex := len(recipes) - 1

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("Name"),
		color.OpBold.Render("Description"),
		color.OpBold.Render("Supported Clouds"),
	)

	tableRow := make([]interface{}, 3)
	for i, r := range recipes {
		tableRow[0] = r.Name

		clouds = []string{}
		for j, c := range r.IaaSList {

			if j == 0 {
				// get description of first recipe/iaas. it
				// is assumed all iaas specific recipes will
				// have the same description
				recipe := config.Config.Context().
					Cookbook().GetRecipe(r.Name, c.Name())
				descLines = utils.SplitString(recipe.Description(), 0, 50, false)
			}
			clouds = append(clouds, c.Name())
		}

		lenDescLines := len(descLines)
		lenClouds := len(clouds)
		if len(descLines) > len(clouds) {
			numLines = lenDescLines
		} else {
			numLines = lenClouds
		}

		for j := 0; j < numLines; j++ {
			if j < lenDescLines {
				tableRow[1] = descLines[j]
			} else {
				tableRow[1] = ""
			}
			if j < lenClouds {
				tableRow[2] = clouds[j]
			} else {
				tableRow[2] = ""
			}
			table.AddRow(tableRow...)
			tableRow[0] = ""
		}
		if i != lastRecipesIndex {
			table.AddSeparator()
		}
	}

	fmt.Printf(
		"\nThis Cloud Builder cookbook supports launching the following recipes.\n\n%s\n",
		table.Render())
}

func ListRecipesForCloud(cloud string) {

	recipes := config.Config.Context().Cookbook().RecipeList()

	table := termtables.CreateTable()
	table.AddHeaders(color.OpBold.Render("Recipe Name"))

	numClouds := 0
	for _, r := range recipes {

		for _, c := range r.IaaSList {

			if c.Name() == cloud {
				numClouds++
				table.AddRow(r.Name)
				break
			}
		}
	}

	if numClouds > 0 {

		fmt.Printf(
			"\nThis Cloud Builder cookbook supports launching the following recipes in the '%s' cloud.\n\n%s\n",
			cloud, table.Render())
	} else {
		fmt.Printf(
			"\nNo recipes found for cloud '%s'.\n",
			cloud)
	}
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false
	flags.StringVarP(&listFlags.cloud, "cloud", "c", "", "list recipes that can be deployed to the given cloud")
}
