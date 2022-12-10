package recipe

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
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

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

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

	for _, r := range cbcli_config.Config.TargetContext().Cookbook().RecipeList() {
		if r.IsBastion {
			spacesRecipes = append(spacesRecipes, r)
		} else {
			appsRecipes = append(appsRecipes, r)
		}
	}

	spacesTable := buildTableOutput(spacesRecipes)
	appsTable := buildTableOutput(appsRecipes)

	fmt.Println("\nThis Cloud Builder configuration supports launching the following recipes.")
	fmt.Println(color.OpBold.Render("\nSpaces\n======\n"))
	if len(spacesRecipes) > 0 {
		fmt.Println(spacesTable.Render())
	} else {
		cbcli_utils.ShowInfoMessage("No space recipes found...\n")
	}
	fmt.Println(color.OpBold.Render("Applications\n============\n"))
	if len(appsRecipes) > 0 {
		fmt.Println(appsTable.Render())
	} else {
		cbcli_utils.ShowInfoMessage("No application recipes found...\n")
	}
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
		color.OpBold.Render("Version"),
		color.OpBold.Render("Description"),
		color.OpBold.Render("Supported Clouds"),
	)

	tableRow := make([]interface{}, 4)
	for i, r := range recipes {
		tableRow[0] = r.RecipeKey
		tableRow[1] = r.CookbookVersion

		clouds = []string{}
		for j, c := range r.IaaSList {

			if j == 0 {
				// get description of first recipe/iaas. it
				// is assumed all iaas specific recipes will
				// have the same description
				recipe := cbcli_config.Config.TargetContext().
					Cookbook().GetRecipe(r.RecipeKey, c.Name())
				descLines = utils.SplitString(recipe.Description(), 0, 50)
			}
			clouds = append(clouds, c.Name())
		}

		lenDescLines := len(descLines)
		lenClouds := len(clouds)
		if lenDescLines > lenClouds {
			numLines = lenDescLines
		} else {
			numLines = lenClouds
		}

		for j := 0; j < numLines; j++ {
			if j < lenDescLines {
				tableRow[2] = descLines[j]
			} else {
				tableRow[2] = ""
			}
			if j < lenClouds {
				tableRow[3] = clouds[j]
			} else {
				tableRow[3] = ""
			}
			table.AddRow(tableRow...)
			tableRow[0] = ""
			tableRow[1] = ""
		}
		if i < lastRecipesIndex {
			table.AddSeparator()
		}
	}

	return table
}

func ListRecipesForCloud(cloud string) {

	recipes := cbcli_config.Config.TargetContext().Cookbook().RecipeList()

	table := termtables.CreateTable()
	table.AddHeaders(color.OpBold.Render("Recipe Name"))

	numClouds := 0
	for _, r := range recipes {

		for _, c := range r.IaaSList {

			if c.Name() == cloud {
				numClouds++
				table.AddRow(r.RecipeKey)
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
