package space

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/eiannone/keyboard"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/userspace"
	"github.com/appbricks/mycloudspace-client/tailscale"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/utils"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var connectFlags = struct {
	commonFlags

	useSpaceDNS    bool
	egressViaSpace bool
}{}

var connectCommand = &cobra.Command{
	Use: "connect [recipe] [cloud] [deployment name]",

	Short: "Connect to the space mesh network.",
	Long: `
Use this command to securely connect to your cloud space's mesh 
netowrk. This enables any device connected to this network to
communicate point-to-point securely and share services and 
applications with each other as well as access space applications
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

func ConnectSpace(space userspace.SpaceNode) {

	var (
		err error

		home    string
		isAdmin bool

		tsd *tailscale.TailscaleDaemon
		tsc *tailscale.TailscaleClient

		cachedIPs []string

		key keyboard.Key

		sent, recd int64
	)

	deviceContext := cbcli_config.Config.DeviceContext()

	if space.GetStatus() != "running" {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Space \"%s\" in \"%s\" region \"%s\" is not online.",
				space.GetSpaceName(), space.GetIaaS(), space.GetRegion(),
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
			space.GetSpaceName(), space.GetIaaS(), space.GetRegion(),
		)
	}
	
	// find home directory.
	home, err = homedir.Dir()
	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}

	// trap keyboard exit/termination event
	disconnect := make(chan bool, 2)

	if runtime.GOOS == "windows" {
		// ctrl-c is not trapped correctly in windows
		// as we also wait on keyboard. this handler
		// traps the event at the win32 API and handles
		// the interrupt to the connection.
		if err = run.HandleInterruptEvent(
			func() bool {
				// ensure all listeners 
				// receive the event
				disconnect <- true
				disconnect <- true
				return true
			},
		); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())	
		}	
	}

	if err := keyboard.Open(); err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	go func() {
		if runtime.GOOS == "windows" {
			// don't handle ctrl-c for windows as that
			// is handled via the interrupt event handler
			for key != keyboard.KeyCtrlX {
				if _, key, err = keyboard.GetKey(); err != nil {
					if err.Error() != "operation canceled" {
						logger.ErrorMessage(
							"ConnectSpace: Unable to pause for key intput. Received error: %s", 
							err.Error(),
						)	
					}
					break
				}
			}	
		} else {
			for key != keyboard.KeyCtrlX && key != keyboard.KeyCtrlC {
				if _, key, err = keyboard.GetKey(); err != nil {
					logger.ErrorMessage(
						"ConnectSpace: Unable to pause for key intput. Received error: %s", 
						err.Error(),
					)
					break
				}
			}	
		}
		// ensure all listeners 
		// receive the event
		disconnect <- true
		disconnect <- true
	}()

	// tailscale daemon starts background network mesh connection services
	tsd = tailscale.NewTailscaleDaemon(
		filepath.Join(home, ".cb", strings.ToLower(deviceContext.GetDevice().Name)), 
		cbcli_config.SpaceNodes, 
		cbcli_config.MonitorService, 
	)
	if cachedIPs, err = tsd.CacheDNSNames(cbcli_config.GetApiEndpointNames()); err != nil {
		logger.ErrorMessage("Error while caching API endpoints in tailscale DNS: %s", err.Error())
	}

	if err = tsd.Start(); err != nil {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf("Error starting space network mesh connection daemon: %s", err.Error()))
	}
	// tailscale client to issue commands to the background service
	tsc = tailscale.NewTailscaleClient(
		tsd.TunnelDeviceName(),
		deviceContext.GetDevice().Name,
		cbcli_config.SpaceNodes,
	)
	tsc.AddSplitDestinations(cachedIPs)
	
	// cleanup on exit
	defer func() {
		_ = keyboard.Close()

		cbcli_config.ShutdownSpinner = spinner.New(
			spinner.CharSets[cbcli_config.SpinnerShutdownType], 
			100*time.Millisecond,
			spinner.WithSuffix(" Shutting down background services."),
			spinner.WithFinalMSG(""),
			spinner.WithHiddenCursor(true),
		)
		cbcli_config.ShutdownSpinner.Start()

		// terminate and cleanup tailscale connection
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
				if err = tsc.Connect(
					space, 
					connectFlags.useSpaceDNS, 
					connectFlags.egressViaSpace,
				); err != nil {
					cbcli_utils.ShowErrorMessage(
						fmt.Sprintf(
							"Failed to initiate login to the space network mesh via the client: %s", 
							err.Error(),
						),
					)
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
		spinner.CharSets[cbcli_config.SpinnerNetworkType], 
		100*time.Millisecond,
		spinner.WithSuffix(" Press CTRL-x or CTRL-c to disconnect."),
		spinner.WithFinalMSG("Connection to space network mesh has been terminated.\n"),
		spinner.WithHiddenCursor(true),
	)

	setStatus := func() {
		status := tsc.GetStatus()
		if status == "Connected" {
			if sent, recd, err = tsd.BytesTransmitted(); err != nil {
				logger.DebugMessage("Error retrieving tailscale connection status: %s", err.Error())
				s.Prefix = "\nUnable to retrieve connection status.\n"
			}
			s.Prefix = fmt.Sprintf(
				"Connected: recd %s, sent %s ", 
				utils.ByteCountIEC(sent), 
				utils.ByteCountIEC(recd),
			)	
		} else {
			s.Prefix = status + " "
		}
	}
	setStatus()
	s.Start()

	for {
		select {
		case <-disconnect:
			s.Stop()
			fmt.Println()
			return
		case <-time.After(time.Millisecond * 500):					
		}
		setStatus()
	}
}

func init() {
	flags := connectCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(connectFlags.commonFlags))

	flags.BoolVarP(&connectFlags.useSpaceDNS, "user-space-dns", "n", false, 
		"use space DNS services")
	flags.BoolVarP(&connectFlags.egressViaSpace, "egress-via-space", "e", false, 
		"egress all network traffic via space node")
}
