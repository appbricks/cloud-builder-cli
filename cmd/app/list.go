package app

import (
	"fmt"

	"github.com/gookit/color"
	"github.com/mevansam/goutils/utils"
	"github.com/mevansam/termtables"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var listCommand = &cobra.Command{
	Use: "list",

	Short: "Lists imported application cookbooks",
	Long: `
Lists all imported cookbooks and their respective recipes.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		ListCookbooks()
	},
	Args: cobra.ExactArgs(0),
}

var showFlags = struct {
	all bool
}{}

func ListCookbooks() {

	table := termtables.CreateTable()
	table.AddHeaders(
		color.OpBold.Render("Name"),
		color.OpBold.Render("Version"),
		color.OpBold.Render("Description"),
		color.OpBold.Render("Recipes"),
	)

	cookbooks := cbcli_config.Config.TargetContext().Cookbook().CookbookList(!showFlags.all)
	lastRecipesIndex := len(cookbooks) - 1

	numRows := 0
	tableRow := make([]interface{}, 4)
	for i, cm := range cookbooks {

		numRows++
		if cm.Imported {
			tableRow[0] = cm.CookbookName
			tableRow[1] = cm.CookbookVersion	
		} else {
			tableRow[0] = color.OpFuzzy.Render(cm.CookbookName)
			tableRow[1] = color.OpFuzzy.Render(cm.CookbookVersion)
		}

		descLines := utils.SplitString(cm.Description, 0, 50)
		lenDescLines := len(descLines)
		lenRecipes := len(cm.Recipes)

		var numLines int
		if lenDescLines > lenRecipes {
			numLines = lenDescLines
		} else {
			numLines = lenRecipes
		}

		for j := 0; j < numLines; j++ {
			if j < lenDescLines {
				if cm.Imported {
					tableRow[2] = descLines[j]
				} else {
					tableRow[2] = color.OpFuzzy.Render(descLines[j])
				}
			} else {
				tableRow[2] = ""
			}
			if j < lenRecipes {
				if cm.Imported {
					tableRow[3] = cm.Recipes[j]
				} else {
					tableRow[3] = color.OpFuzzy.Render(cm.Recipes[j])
				}
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
	
	if numRows > 0 {
		if showFlags.all {
			fmt.Println("\nThis Cloud Builder configuration contains the following cookbooks.")
		} else {
			fmt.Println("\nThis Cloud Builder configuration contains the following imported cookbooks.")
		}		
		fmt.Println()
		fmt.Println(table.Render())

	} else {
		cbcli_utils.ShowInfoMessage("\nNo imported cookbooks found...\n")
	}
}

func init() {
	flags := listCommand.Flags()
	flags.SortFlags = false

	flags.BoolVarP(&showFlags.all, "all", "a", false, "show all cookbooks")
}
