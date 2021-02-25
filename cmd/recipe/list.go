package recipe

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
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
		spacesRecipes,
		appsRecipes []cookbook.CookbookRecipeInfo
	)

	for _, r := range cbcli_config.Config.Context().Cookbook().RecipeList() {
		if r.IsBastion {
			spacesRecipes = append(spacesRecipes, r)
		} else {
			appsRecipes = append(appsRecipes, r)
		}
	}

	spacesTable := buildTableOutput(spacesRecipes)
	appsTable := buildTableOutput(appsRecipes)

	fmt.Println("\nThis Cloud Builder cookbook supports launching the following recipes.")
	fmt.Println(color.OpBold.Render("\nSpaces\n======\n"))
	if len(spacesRecipes) > 0 {
		fmt.Println(spacesTable.Render())
	} else {
		cbcli_utils.ShowInfoMessage("No space recipes found...")
	}
	fmt.Println(color.OpBold.Render("Applications\n============\n"))
	if len(appsRecipes) > 0 {
		fmt.Println(appsTable.Render())
	} else {
		cbcli_utils.ShowInfoMessage("No application recipes found...")
	}
	fmt.Println()
}

func buildTableOutput(recipes []cookbook.CookbookRecipeInfo) *termtables.Table {

	var (
		clouds, 
		descLines []string

		numLines int
	)

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
				recipe := cbcli_config.Config.Context().
					Cookbook().GetRecipe(r.Name, c.Name())
				descLines = utils.SplitString(recipe.Description(), 0, 50)
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
		if i < lastRecipesIndex {
			table.AddSeparator()
		}
	}

	return table
}

func ListRecipesForCloud(cloud string) {

	recipes := cbcli_config.Config.Context().Cookbook().RecipeList()

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
