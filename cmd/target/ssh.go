package target

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/goutils/streams"
	"github.com/mevansam/goutils/utils"

	"github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var sshFlags = struct {
	sudo bool
}{}

var sshCommand = &cobra.Command{
	Use: "ssh [recipe] [cloud] [region] [deployment name] [instance name]",

	Short: "SSH to a launch target's resource.",
	Long: `
SSH the given target's named resource. This is for advance users as
well as for troubleshooting any configuration errors at the target.
This sub-command can be run only on instances that have a public IP
and allow SSH ingress from the internet. If the instance is internal
then this command can only be run once the VPN connection to the 
cloud space sandbox VPN has been establised.
`,

	Run: func(cmd *cobra.Command, args []string) {
		SSHTarget(args[0], args[1], args[2], args[3], args[4])
	},
	Args: cobra.ExactArgs(5),
}

func SSHTarget(recipe, cloud, region, deploymentName, instanceName string) {

	var (
		err error

		tgt    *target.Target
		client *utils.SSHClient
	)

	targets := config.Config.Context().TargetSet()
	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, cloud, region, deploymentName)
	if tgt = targets.GetTarget(targetName); tgt != nil {

		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		managedInstance := tgt.ManagedInstance(instanceName)

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
	}
}

func StartTerminal(client *utils.SSHClient, rootPassword string) error {

	var (
		err error

		osStdInFd             int
		origTermState         *terminal.State
		termWidth, termHeight int

		sshTermConfig *utils.SSHTerminalConfig

		expectStream *streams.ExpectStream
		stdinSender  io.ReadCloser
		stdoutSender io.WriteCloser
	)

	osStdInFd = int(os.Stderr.Fd())
	if terminal.IsTerminal(osStdInFd) {
		if origTermState, err = terminal.MakeRaw(osStdInFd); err != nil {
			return err
		}
		defer terminal.Restore(osStdInFd, origTermState)

		if termWidth, termHeight, err = terminal.GetSize(osStdInFd); err != nil {
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
		expectStream.AddExpectOutTrigger(
			&streams.Expect{
				StartPattern: `password for [a-z_][a-z0-9_-]*`,
				Command:      rootPassword + "\n",
			},
			true,
		)
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
	flags.BoolVarP(&sshFlags.sudo, "sudo", "s", false, "sudo to root shell after establishing the SSH session")
}
