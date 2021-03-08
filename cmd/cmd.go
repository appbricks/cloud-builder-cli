package cmd

// Copyright © 2019 Mevan Samaratunga
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gookit/color"
	"github.com/peterh/liner"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"

	"github.com/appbricks/cloud-builder-cli/cmd/cloud"
	"github.com/appbricks/cloud-builder-cli/cmd/initialize"
	"github.com/appbricks/cloud-builder-cli/cmd/recipe"
	"github.com/appbricks/cloud-builder-cli/cmd/target"
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/cookbook"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_cookbook "github.com/appbricks/cloud-builder-cli/cookbook"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use: "cb",

	Short: "Use the cli to launch secure services in the cloud.",
	Long: `
The Cloud Builder CLI can be used to install applications and
services to a virtual network sandbox in your public cloud account.
This cloud space is secured behind a VPN service created when
launching this sandbox in a public cloud region. Once the VPN has
been established, all traffic from your devices to applications and
services within the sandbox as well as the internet will pass through
this encrypted tunnel. This ensures all network traffic from your
personal devices, the sandbox and the internet is secured and
anonymized as it traverses the public provider networks.
`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if !cbcli_config.Config.EULAAccepted() {
			fmt.Println(`
Before you can deploy Cloud Builder recipes you need to review and
accept the AppBricks, Inc. Software End User Agreement. The terms of
the agreement can be found at the following link.

` + color.FgBlue.Render(`https://appbricks.io/legal/`))

			response := cbcli_utils.GetUserInputFromList(
				"Do you agree to the terms: ",
				"yes",
				[]string{"no", "yes"},
			)
			if response == "yes" {
				cbcli_config.Config.SetEULAAccepted()
			} else {
				cbcli_utils.ShowErrorAndExit("You need to accept the EULA to use the Cloud Builder CLI.")
			}
		}

		if cmd != initialize.InitCommand {
			if !cbcli_config.Config.Initialized() {
				fmt.Println(
					color.OpReverse.Render(
						color.Yellow.Render(
							"\n>> Please initialize the Cloud Builder client to secure configuration settings.",
						),
					),
				)	
			} else {
				if err := cbcli_auth.Authenticate(cbcli_config.Config); err != nil {					
					cbcli_utils.ShowErrorAndExit("My Cloud Space user authentication failed.")
				}
				if err := cbcli_auth.ValidateAuthenticatedUser(cbcli_config.Config); err != nil {					
					cbcli_utils.ShowErrorAndExit("My Cloud Space authenticated user validation failed.")
				}
			}
		}
	},
}

func Execute() {

	var (
		err error
	)

	logger.Initialize()

	if err = rootCmd.Execute(); err != nil {
		logger.TraceMessage("Command Execute() returned and error: ", err.Error())
		os.Exit(1)
	}

	if cbcli_config.Config != nil {		
		if admin, _ := run.IsAdmin(); !admin {
			if err = cbcli_config.Config.Save(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}	
		} else {
			logger.TraceMessage("Config has not been saved as CLI was run as root or with elevated privileges.")
		}
	}
}

func init() {

	var (
		err error

		home string
	)

	// handle ctrl+c
	setupCloseHandler()

	if systemPassphrase := os.Getenv("CBS_SYSTEM_PASSPHRASE"); len(systemPassphrase) > 0 {
		config.SystemPassphrase = func() string {
			return systemPassphrase
		}
	}

	cobra.OnInitialize(initConfig)
	cobra.EnableCommandSorting = false
	addCommands()

	// find home directory.
	home, err = homedir.Dir()
	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", home+"/.cb/config.yml", "config file")
}

// read in config file and ENV variables if set.
func initConfig() {

	var (
		err error

		cbCookbook *cookbook.Cookbook
	)

	// load embedded cookbook
	if cbCookbook, err = cbcli_cookbook.NewCookbook(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	// initialize / load config file
	if cbcli_config.Config, err = config.InitFileConfig(cfgFile, cbCookbook, getPassphrase); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if err = cbcli_config.Config.Load(); err != nil {
		logger.DebugMessage("Error loading the configuration: %s", err.Error())

		fmt.Println("Failed to unlock configuration file!")
		os.Exit(1)
	}
}

// get encryption passphrase from use input
func getPassphrase() string {

	var (
		err error

		passphrase string
	)

	line := liner.NewLiner()
	defer func() {
		line.Close()
		if err := recover(); err != nil {
			if err == liner.ErrPromptAborted {
				fmt.Println("\nInitialization aborted.")
				os.Exit(1)
			} else {
				cbcli_utils.ShowErrorAndExit(err.(error).Error())
				os.Exit(1)
			}
		}
	}()

	line.SetCtrlCAborts(true)

	fmt.Println()
	if passphrase, err = line.PasswordPrompt("Please enter the passphrase to unlock the configuration : "); err != nil {
		panic(err)
	}
	return passphrase
}

func setupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println(
			color.Red.Render(
				"\n\nCLI command execution has been interrupted.",
			),
		)
		os.Exit(1)
	}()
}

// adds commands to the root
func addCommands() {
	rootCmd.AddCommand(versionCommand)
	rootCmd.AddCommand(initialize.InitCommand)
	rootCmd.AddCommand(cloud.CloudCommands)
	rootCmd.AddCommand(recipe.RecipeCommands)
	rootCmd.AddCommand(target.TargetCommands)
}
