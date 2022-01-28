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
	"path/filepath"
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
	"github.com/appbricks/cloud-builder-cli/cmd/space"
	"github.com/appbricks/cloud-builder-cli/cmd/target"
	"github.com/appbricks/cloud-builder/config"
	"github.com/appbricks/cloud-builder/cookbook"
	"github.com/appbricks/mycloudspace-client/mycscloud"
	"github.com/appbricks/mycloudspace-common/monitors"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_cookbook "github.com/appbricks/cloud-builder-cli/cookbook"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var (
	isProd  string
	cfgFile string
)

// commands that do not require
// authenticated access
var noauthCmds = map[string]bool{
	"help": true,
	"version": true,
	"init": true,
	"logout": true,
}

var spaceCmds = map[string]bool{
	"target": true,
	"space": true,
}

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
			fmt.Println()
			cbcli_utils.ShowMessage(
				"Before you can deploy Cloud Builder recipes you need to review and" +
				"accept the AppBricks, Inc. Software End User Agreement. The terms of" +
				"the agreement can be found at the following link.",
			)
			fmt.Println()
			fmt.Println(color.FgBlue.Render(`https://appbricks.io/legal/`))

			response := cbcli_utils.GetUserInputFromList(
				"Do you agree to the terms: ",
				"yes",
				[]string{"no", "yes"},
				true,
			)
			if response == "yes" {
				cbcli_config.Config.SetEULAAccepted()
			} else {
				cbcli_utils.ShowErrorAndExit("You need to accept the EULA to use the Cloud Builder CLI.")
			}
		}

		// retrieve the command
		// to check against
		cmdName := cmd.Name()
		if cmd.Parent() != nil && cmd.Parent().Parent() != nil {
			cmdName = cmd.Parent().Name()
		}

		if _, noauth := noauthCmds[cmdName]; !noauth {
			if !cbcli_config.Config.Initialized() {
				fmt.Println(
					color.OpReverse.Render(
						color.Yellow.Render(
							"\n>> Please initialize the Cloud Builder client to secure configuration settings.",
						),
					),
				)
				// load space target nodes if 
				// executing a target command
				if cmd.Parent() != nil && cmd.Parent().Name() == "target" {
					cbcli_config.SpaceNodes = mycscloud.NewSpaceNodes(cbcli_config.Config)
				}

			} else {

				var (
					err error

					awsAuth *cbcli_auth.AWSCognitoJWT
				)
				if awsAuth, err = cbcli_auth.GetAuthenticatedToken(cbcli_config.Config, false); err != nil {
					logger.DebugMessage("Authentication returned error: %s", err.Error())
					cbcli_utils.ShowErrorAndExit("My Cloud Space user authentication failed.")
				}
				if err = cbcli_auth.AuthorizeDeviceAndUser(cbcli_config.Config); err != nil {
					logger.DebugMessage("Authorizing logged in user on this device returned error: %s", err.Error())
					cbcli_utils.ShowErrorAndExit("My Cloud Space device and user authorization failed.")
				}
				if !cbcli_config.Config.DeviceContext().IsAuthorizedUser(awsAuth.Username()) {
					// reset command
					cmd.Run = func(cmd *cobra.Command, args []string) {}
				} else {
					// show logged in message only if cli 
					// is being run via a non-root user				
					if isAdmin, _ := run.IsAdmin(); !isAdmin {
						fmt.Println()
						cbcli_utils.ShowNoticeMessage("You are logged in as \"%s\".", awsAuth.Username())	
					}
					// load space target nodes if 
					// executing a target command
					if _, isSpaceCmd := spaceCmds[cmdName]; isSpaceCmd {
						if cbcli_config.SpaceNodes, err = mycscloud.GetSpaceNodes(cbcli_config.Config, cbcli_config.AWS_USERSPACE_API_URL); err != nil {
							logger.DebugMessage("Failed to load and merge remote space nodes with local targets: %s", err.Error())
							cbcli_utils.ShowErrorAndExit("Failed to load user's space nodes.")
						}								
					}
				}
			}
		}
	},
}

func Execute() {

	var (
		err error
	)

	defer func() {
		if cbcli_config.MonitorService != nil {
			cbcli_config.MonitorService.Stop()
		}
		if cbcli_config.ShutdownSpinner != nil {
			cbcli_config.ShutdownSpinner.Stop()
		}
	}()

	if isProd == "yes" {
		logLevel := os.Getenv("CBS_LOGLEVEL")
		if len(logLevel) == 0 {
			// default is error but for prod builds we do 
			// not want to show errors unless requested
			os.Setenv("CBS_LOGLEVEL", "fatal")

		} else if logLevel == "trace" {
			// reset trace log level if set for prod builds
			cbcli_utils.ShowWarningMessage(
				"Trace log-level is not supported in prod build. Resetting level to 'debug'.\n",
			)
			os.Setenv("CBS_LOGLEVEL", "debug")
		}
	}
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", filepath.Join(home, ".cb", "config.yml"), "config file")
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

	eventPublisher := mycscloud.NewEventPublisher(cbcli_config.AWS_USERSPACE_API_URL, "", cbcli_config.Config)
	cbcli_config.MonitorService = monitors.NewMonitorService(eventPublisher, 1 /* publish monitor events to cloud every 1s */)
	if err = cbcli_config.MonitorService.Start(); err != nil {
		logger.DebugMessage("Failed to start monitor service: %s", err.Error())

		fmt.Println("Failed to start internal telemetry services!")
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
	rootCmd.AddCommand(logoutCommand)
	rootCmd.AddCommand(cloud.CloudCommands)
	rootCmd.AddCommand(recipe.RecipeCommands)
	rootCmd.AddCommand(target.TargetCommands)
	rootCmd.AddCommand(space.SpaceCommands)
}
