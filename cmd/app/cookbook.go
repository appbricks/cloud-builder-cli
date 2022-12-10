package app

import (
	"github.com/spf13/cobra"
)

var CookbookCommands = &cobra.Command{
	Use: "cookbook",

	Short: "Import, delete, list cookbooks.",
	Long: `
Imports, deletes and lists application cookbooks. Cookbooks
are a collections of similar recipes for building application
infrastructure and installing applications to a cloud space. 
`,
}

func init() {
	CookbookCommands.AddCommand(importCommand)
	CookbookCommands.AddCommand(deleteCommand)
	CookbookCommands.AddCommand(listCommand)
}
