package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/mevansam/goutils/logger"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var importCommand = &cobra.Command{
	Use: "import [cookbook file path]",

	Short: "Import recipes",
	Long: `
Imports recipes from an application cookbook zip file downloaded from
the MyCloudSpace app store. 
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		ImportRecipe(args)
	},
	Args: cobra.RangeArgs(0, 1),
}

func ImportRecipe(args []string) {

	var (
		err error

		importZipFile string

		fi os.FileInfo
	)

	if len(args) == 0 {
		fmt.Println()
		importZipFile = strings.Trim(
			cbcli_utils.GetUserInput("Path to cookbook zip file to import (you can drag/drop from a finder/explorer window to the terminal) : "),
			" '\"",
		)

	} else {
		importZipFile = args[0]
	}

	fi, err = os.Stat(importZipFile)
	if os.IsNotExist(err) || fi.IsDir() {
		cbcli_utils.ShowErrorAndExit(fmt.Sprintf("%s is not a valid file.", importZipFile))
	}
	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	fmt.Println()
	workingSpinner := spinner.New(
		spinner.CharSets[cbcli_config.SpinnerWorking], 
		100*time.Millisecond,
		spinner.WithSuffix(" Importing cookbook."),
		spinner.WithFinalMSG(""),
		spinner.WithHiddenCursor(true),
	)
	workingSpinner.Start()	

	if err = cbcli_config.Config.TargetContext().Cookbook().ImportCookbook(importZipFile); err != nil {
		workingSpinner.Stop()

		logger.ErrorMessage("Failed to import cookbook zip file: %s", err.Error())
		cbcli_utils.ShowErrorAndExit(fmt.Sprintf("Invalid cookbook zip archive %s.", importZipFile))

	} else {
		workingSpinner.Stop()
	}
}

func init() {
	flags := importCommand.Flags()
	flags.SortFlags = false
}
