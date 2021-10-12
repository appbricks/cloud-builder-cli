package space

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/eiannone/keyboard"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/tailscale"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var connectFlags = struct {
	commonFlags

	// superUser bool
	// download  bool
}{}

var connectCommand = &cobra.Command{
	Use: "connect [recipe] [cloud] [deployment name]",

	Short: "Connect to the space mesh network.",
	Long: `
Use this command to securely connect to your cloud space's mesh 
netowrk. This enables any device connected to this network to
communicate point-to-point securely and share services and 
applications with each other as well as accessing space applications
that have been shared with the mesh. Shared applications and
permissions can be managed via the MyCloudSpace account and space
management dashboard.
`,

	PreRun: authorizeSpaceNode(auth.NewRoleMask(auth.Admin, auth.Manager, auth.Guest), &(connectFlags.commonFlags)),

	Run: func(cmd *cobra.Command, args []string) {
		ConnectSpace(spaceNode)
	},
	Args: cobra.ExactArgs(3),
}

func ConnectSpace(spaceNode userspace.SpaceNode) {

	var (
		err error

		home    string
		isAdmin bool

		tsd *tailscale.TailscaleDaemon

		key keyboard.Key
	)

	if spaceNode.GetStatus() != "running" {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Space \"%s\" in \"%s\" region \"%s\" is not online.",
				spaceNode.GetSpaceName(), spaceNode.GetIaaS(), spaceNode.GetRegion(),
			),
		)
		// TODO: if the current logged in user has admin access he should be able to resume the space
	}
	
	// re-spawn the CLI with elevated privileges 
	// if it is not running cli with such access
	if isAdmin, err = run.IsAdmin(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if !isAdmin {
		cbcli_utils.ShowWarningMessage("\nPlease enter you password for admin privileges required to update the network configuration, if requested.")
		if err = run.RunAsAdmin(os.Stdout, os.Stderr); err != nil {
			logger.DebugMessage(
				"Execution of CLI command with elevated privileges failed with error: %s", 
				err.Error(),
			)
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	} else {
		cbcli_utils.ShowInfoMessage(
			"\nConnecting to space \"%s\" in \"%s\" region \"%s\".",
			spaceNode.GetSpaceName(), spaceNode.GetIaaS(), spaceNode.GetRegion(),
		)
	}
	
	// find home directory.
	home, err = homedir.Dir()
	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	// tailscale daemon starts background network mesh connection services
	tsd = tailscale.NewTailscaleDaemon(
		cbcli_config.Config, cbcli_config.SpaceNodes, filepath.Join(home, ".cb"),
	)
	if err = tsd.Start(); err != nil {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf("Error starting space network mesh connection daemon: %s", err.Error()))
	}
	// tailscale client to issue commands to the background service
	tsc := tailscale.NewTailscaleClient(spaceNode)	
	
	// trap keyboard exit/termination event
	disconnect := make(chan bool)
	if err := keyboard.Open(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	go func() {
		for key != keyboard.KeyCtrlX && key != keyboard.KeyCtrlC {
			if _, key, err = keyboard.GetKey(); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		}
		disconnect <- true
	}()

	// cleanup on exit
	defer func() {
		_ = keyboard.Close()
		if err = tsc.Disconnect(); err != nil {
			logger.DebugMessage("Error disconnecting tailscale client: %s", err.Error())
		}
		tsd.Stop()
	}()
	
	// intitiate the connecting to the space network. 
	// timeout after 5s if connection cannot be established
	retryTimer := time.NewTicker(500 * time.Millisecond)	
	go func() {
		defer retryTimer.Stop()

		for timeoutCounter := 0;
			timeoutCounter < 10; 
			timeoutCounter++ {

			select {
			case <-disconnect:
				return
			case <-retryTimer.C:
				if err = tsc.Connect(); err != nil {
					logger.TraceMessage(
						fmt.Sprintf("Failed to initiate login to the space network mesh via the client: %s", err.Error()))
				} else {
					return
				}	
			}
		}
		cbcli_utils.ShowErrorMessage("Timed out while attempting to connect to space network mesh.")
		disconnect <- true
	}()

	fmt.Println()
	s := spinner.New(
		spinner.CharSets[39], 
		100*time.Millisecond,
		spinner.WithSuffix(" Press CTRL-x or CTRL-c to disconnect."),
		spinner.WithFinalMSG("Connection to space network mesh has been terminated.\n"),
		spinner.WithHiddenCursor(true),
	)
	s.Prefix = tsc.GetStatus() + " "
	s.Start()

	for {
		select {
		case <-disconnect:
			s.Stop()
			fmt.Println()
			return
		case <-time.After(time.Millisecond * 100):					
		}
		s.Prefix = tsc.GetStatus() + " "
	}
}

func init() {
	flags := connectCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(connectFlags.commonFlags))

	// flags.BoolVarP(&connectFlags.superUser, "super", "u", false, 
	// 	"connect as a super user with no restrictions")
	// flags.BoolVarP(&connectFlags.download, "download", "d", false, 
	// 	"download the VPN configuration file instead of\nestablishing a connection")
}
