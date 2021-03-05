package target

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/streams"
	"github.com/mevansam/goutils/utils"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var sshFlags = struct {
	commonFlags

	sudo bool
}{}

var sshCommand = &cobra.Command{
	Use: "ssh [recipe] [cloud] [deployment name]",

	Short: "SSH to a launch target's resource.",
	Long: `
Create an SSH session to one of the target's running instance
resources. This sub-command is available for advance users as well as
for troubleshooting any configuration errors at the target. This sub-
command can be run only on instances that are running, have a public
IP and allow SSH ingress from the internet. If the instance is
internal then this command can only be run once the VPN connection to
the cloud space sandbox VPN has been establised.
`,

	Run: func(cmd *cobra.Command, args []string) {
		SSHTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(sshFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func SSHTarget(targetKey string) {

	var (
		err error

		response string

		tgt    *target.Target
		state  cloud.InstanceState
		client *utils.SSHClient

		managedInstance *target.ManagedInstance
		instanceIndex   int
	)

	targets := cbcli_config.Config.Context().TargetSet()
	if tgt = targets.GetTarget(targetKey); tgt != nil {

		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}

		managedInstances := tgt.ManagedInstances()
		numInstances := len(managedInstances)
		if numInstances > 1 {

			options := make([]string, numInstances)

			fmt.Print("\nTarget is running more than one managed instance given below.\n\n")
			for i, mi := range managedInstances {
				option := strconv.Itoa(i + 1)
				fmt.Print(color.Green.Render(option))
				fmt.Println(" - " + mi.Name())
				options[i] = option
			}
			fmt.Println()

			if response = cbcli_utils.GetUserInputFromList(
				"Enter # of instance to SSH to or (q)uit: ",
				"", options); response == "q" {
				return
			}
			if instanceIndex, err = strconv.Atoi(response); err != nil ||
				instanceIndex < 1 || instanceIndex > numInstances {
				cbcli_utils.ShowErrorAndExit("invalid option provided")
			}
			instanceIndex--
			managedInstance = managedInstances[instanceIndex]

		} else {
			managedInstance = managedInstances[0]
		}

		if state, err = managedInstance.Instance.State(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		if state == cloud.StateRunning {
			if client, err = utils.SSHDialWithKey(
				managedInstance.SSHAddress(),
				managedInstance.SSHUser(),
				managedInstance.SSHKey(),
			); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			defer client.Close()

			if err = StartTerminal(client, managedInstance.RootPassword()); err != nil {
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
		} else {
			cbcli_utils.ShowErrorAndExit("instance is not running")
		}
	
		return
	}

	cbcli_utils.ShowErrorAndExit(
		fmt.Sprintf(
			"Target \"%s\" does not exist. Run 'cb target list' to list the currently configured targets",
			targetKey,
		),
	)
}

func StartTerminal(client *utils.SSHClient, rootPassword string) error {

	var (
		err error

		osStdinFd             int
		origTermState         *term.State
		termWidth, termHeight int

		sshTermConfig *utils.SSHTerminalConfig

		expectStream *streams.ExpectStream
		stdinSender  io.ReadCloser
		stdoutSender io.WriteCloser
	)

	osStdinFd = int(os.Stdin.Fd())
	if term.IsTerminal(osStdinFd) {
		if origTermState, err = term.MakeRaw(osStdinFd); err != nil {
			return err
		}
		defer func() {
			if err = term.Restore(osStdinFd, origTermState); err != nil {
				logger.TraceMessage(
					"Error restoring streams: %s", err.Error(),
				)
			}
		}()

		if termWidth, termHeight, err = term.GetSize(osStdinFd); err != nil {
			return err
		}
		sshTermConfig = &utils.SSHTerminalConfig{
			Term:   "xterm-256color",
			Height: termHeight,
			Weight: termWidth,
			Modes: ssh.TerminalModes{
				ssh.ECHO:          1,     // enable echoing
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			},
		}

	} else {
		sshTermConfig = &utils.SSHTerminalConfig{
			Term:   "xterm",
			Height: 40,
			Weight: 80,
			Modes: ssh.TerminalModes{
				ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
				ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
			},
		}
	}

	if sshFlags.sudo {
		expectStream, stdinSender, stdoutSender = streams.NewExpectStream(
			os.Stdin, os.Stdout, func() {
				client.Close()
			},
		)
		defer expectStream.Close()

		expectStream.AddExpectOutTrigger(
			&streams.Expect{
				StartPattern: `^Welcome to Ubuntu`,
				EndPattern:   `[a-z_][a-z0-9_-]*@.*:.*\~.*\$`,
				Command:      "sudo su -\n",
			},
			true,
		)
		if len(rootPassword) > 0 {
			expectStream.AddExpectOutTrigger(
				&streams.Expect{
					StartPattern: `password for [a-z_][a-z0-9_-]*`,
					Command:      rootPassword + "\n",
				},
				true,
			)	
		}
		expectStream.StartAsShell()

		if err := client.
			Terminal(sshTermConfig).
			SetStdio(stdinSender, stdoutSender, stdoutSender).
			Start(); err != nil {

			// ignore and exit error
			return nil
		}

	} else {
		if err := client.
			Terminal(sshTermConfig).
			SetStdio(os.Stdin, os.Stdout, os.Stderr).
			Start(); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	flags := sshCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(sshFlags.commonFlags))

	flags.BoolVarP(&sshFlags.sudo, "sudo", "u", false, 
		"sudo to root shell after establishing the SSH session")
}
