package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const VERSION = `0.0.0`
const BUILD_TIMESTAMP = `November 19, 2021 at 09:16 EST`

var versionCommand = &cobra.Command{
	Use: "version",

	Short: "Show the version of the Cloud Builder CLI.",
	Long: `
Display the version of the Cloud Builder CLI and any other relevant
configuration status.
`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("\nVersion:    %s\n", VERSION)
		fmt.Printf("Build date: %s\n", BUILD_TIMESTAMP)
	},
}
