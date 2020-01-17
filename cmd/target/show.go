package target

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/mevansam/gocloud/cloud"
	"github.com/mevansam/goforms/forms"
	"github.com/mevansam/goforms/ux"
	"github.com/mevansam/goutils/term"
	"github.com/mevansam/goutils/utils"

	"github.com/appbricks/cloud-builder-cli/config"
	"github.com/appbricks/cloud-builder/target"

	cbcli_utils "github.com/appbricks/cloud-builder-cli/utils"
)

var showFlags = struct {
	config bool
	all    bool
}{}

var showCommand = &cobra.Command{
	Use: "show [recipe] [cloud] [region] [deployment name]",

	Short: "Show configuration data for a target.",
	Long: `
Show the deployment configuration values for the target. If the
target has not been created and configured then this sub-command will
return an error. Run 'cb target list' to view the list of configured
targets.
`,

	Run: func(cmd *cobra.Command, args []string) {
		ShowRecipe(args[0], args[1], args[2], args[3])
	},
	Args: cobra.ExactArgs(4),
}

func ShowRecipe(recipe, cloud, region, deploymentName string) {

	var (
		err error

		tgt       *target.Target
		inputForm forms.InputForm
	)

	targetName := fmt.Sprintf("%s/%s/%s/%s", recipe, cloud, region, deploymentName)
	if tgt, err = config.Config.Context().GetTarget(targetName); err == nil && tgt != nil {

		if err = tgt.LoadRemoteRefs(); err != nil {
			cbcli_utils.ShowErrorAndExit(err.Error())
		}
		showNodeInfo(tgt)

		if showFlags.config || showFlags.all {
			if inputForm, err = tgt.Provider.InputForm(); err != nil {
				// if this happens there is an internal
				// error and it is most likely a bug
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showInputFormData(
				fmt.Sprintf("\nProvider Configuration for Target \"%s\"", targetName),
				inputForm,
			)
			if inputForm, err = tgt.Recipe.InputForm(); err != nil {
				// if this happens there is an internal
				// error and it is most likely a bug
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showInputFormData(
				fmt.Sprintf("Recipe Configuration for Target \"%s\"", targetName),
				inputForm,
			)
			if inputForm, err = tgt.Backend.InputForm(); err != nil {
				// if this happens there is an internal
				// error and it is most likely a bug
				cbcli_utils.ShowErrorAndExit(err.Error())
			}
			showInputFormData(
				fmt.Sprintf("Backend Configuration for Target \"%s\"", targetName),
				inputForm,
			)
		}

		return
	}

	if err != nil {
		cbcli_utils.ShowErrorAndExit(err.Error())
	} else {
		cbcli_utils.ShowErrorAndExit(
			fmt.Sprintf(
				"Unknown target named \"%s\". Run 'cb target list' "+
					"to list the currently configured targets",
				targetName,
			),
		)
	}
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
	flags.BoolVarP(&showFlags.all, "config", "c", false, "show required configuration data values")
	flags.BoolVarP(&showFlags.all, "all", "a", false, "show all configuration data values")
}
