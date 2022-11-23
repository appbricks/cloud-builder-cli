package target

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/appbricks/cloud-builder/target"
	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/term"
	"github.com/mevansam/goutils/utils"

	cbcli_auth "github.com/appbricks/cloud-builder-cli/auth"
	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var showFlags = struct {
	commonFlags

	config bool
	all    bool
}{}

var showCommand = &cobra.Command{
	Use: "show [recipe] [cloud] [deployment name]",

	Short: "Show configuration data for a target.",
	Long: `
Show the deployment configuration values for the target. If the
target has not been created and configured then this sub-command will
return an error. Run 'cb target list' to view the list of configured
targets.
`,

	PreRun: cbcli_auth.AssertAuthorized(auth.NewRoleMask(auth.Admin), nil),

	Run: func(cmd *cobra.Command, args []string) {
		ShowTarget(getTargetKeyFromArgs(args[0], args[1], args[2], &(showFlags.commonFlags)))
	},
	Args: cobra.ExactArgs(3),
}

func ShowTarget(targetKey string) {

	var (
		err error

		tgt       *target.Target
		inputForm forms.InputForm
	)

	if tgt, err = cbcli_config.Config.TargetContext().GetTarget(targetKey); err == nil && tgt != nil {

		showNodeInfo(tgt)

		if showFlags.config || showFlags.all {
			if inputForm, err = tgt.Provider.InputForm(); err != nil {
				// if this happens there is an internal
				// error and it is most likely a bug
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showInputFormData(
				fmt.Sprintf("Provider Configuration for Target \"%s\"", tgt.DeploymentName()),
				inputForm,
			)
			if inputForm, err = tgt.Recipe.InputForm(); err != nil {
				// if this happens there is an internal
				// error and it is most likely a bug
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showInputFormData(
				fmt.Sprintf("Recipe Configuration for Target \"%s\"", tgt.DeploymentName()),
				inputForm,
			)
			if inputForm, err = tgt.Backend.InputForm(); err != nil {
				// if this happens there is an internal
				// error and it is most likely a bug
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showInputFormData(
				fmt.Sprintf("Backend Configuration for Target \"%s\"", tgt.DeploymentName()),
				inputForm,
			)
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

func showNodeInfo(tgt *target.Target) {

	var (
		err  error
		l    int
		text strings.Builder

		state cloud.InstanceState
	)

	text.WriteString("\nDeployment: ")
	text.WriteString(strings.ToUpper(tgt.DeploymentName()))
	l = text.Len()
	text.Write(term.LineFeedB)
	utils.RepeatString("=", l, &text)
	fmt.Println(color.OpBold.Render(text.String()))

	fmt.Print("\nStatus: ")
	fmt.Println(getTargetStatusName(tgt))

	fmt.Println()
	fmt.Print(tgt.Description())

	for _, managedInstance := range tgt.ManagedInstances() {

		text.Reset()
		text.WriteString("\nInstance: ")
		text.WriteString(managedInstance.Name())
		l = text.Len()
		text.Write(term.LineFeedB)
		utils.RepeatString("-", l, &text)
		fmt.Println(color.OpBold.Render(text.String()))

		fmt.Print("\nState: ")
		if state, err = managedInstance.Instance.State(); err == nil {
			switch state {
			case cloud.StateRunning:
				fmt.Println(
					color.OpReverse.Render(
						color.Green.Render("running"),
					),
				)
			case cloud.StateStopped:
				fmt.Println(
					color.OpReverse.Render(
						color.Red.Render("stopped"),
					),
				)
			case cloud.StatePending:
				fmt.Println(
					color.OpReverse.Render(
						color.Yellow.Render("pending"),
					),
				)
			default:
				fmt.Println("Unknown")
			}
		}

		fmt.Println()
		fmt.Print(managedInstance.Description())
		fmt.Println()
	}
}

func showInputFormData(title string, inputForm forms.InputForm) {

	var (
		err error

		textForm *ux.TextForm
	)

	fmt.Printf("\n")
	if textForm, err = ux.NewTextForm(
		title,
		"CONFIGURATION DATA",
		inputForm); err != nil {
		// if this happens there is an internal
		// error and it is most likely a bug
		cbcli_utils.ShowErrorAndExit(err.Error())
	}
	if showFlags.all {
		textForm.ShowInputReference(
			ux.DescAndValues,
			0, 2, 80)
	} else {
		textForm.ShowInputReference(
			ux.DescAndValues,
			0, 2, 80,
			"target-undeployed", "target-deployed",
		)
	}
	fmt.Print("\n\n")
}

func init() {
	flags := showCommand.Flags()
	flags.SortFlags = false
	bindCommonFlags(flags, &(showFlags.commonFlags))	

	flags.BoolVarP(&showFlags.config, "config", "c", false, 
		"show required configuration data values")
	flags.BoolVarP(&showFlags.all, "all", "a", false, 
		"show all configuration data values")
}
