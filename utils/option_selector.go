package utils

import (
	"fmt"
	"strconv"

	"github.com/appbricks/cloud-builder/auth"
	"github.com/gookit/color"
)

type OptionSelector struct {
	Options []Option

	OptionListFilter map[string][]int
	OptionRoleFilter map[auth.Role]map[int]bool
}

type Option struct {
	Text    string
	Command func(data interface{}) error
}

func (os OptionSelector) SelectOption(data interface{}, listFilter string, roleFilter auth.Role) error {

	var (
		err error

		response string
		selected int
	)

	enabledOptions := os.OptionListFilter[listFilter]
	numEnabledOptions := len(enabledOptions)
	for i, c := range os.Options {
		if os.optionAllowedInRole(roleFilter, enabledOptions, i) {
			fmt.Print(color.Green.Render(strconv.Itoa(i + 1)))
			fmt.Println(c.Text)
		} else {
			fmt.Println(color.OpFuzzy.Render(strconv.Itoa(i+1) + c.Text))
		}
	}
	fmt.Println()

	optionList := make([]string, numEnabledOptions)
	allowedOptions := make(map[int]bool)
	for i, o := range enabledOptions {
		o++
		optionList[i] = strconv.Itoa(o)
		allowedOptions[o] = true
	}
	if response = GetUserInputFromList(
		"Enter # of option or (q)uit: ",
		"", optionList, false); response == "q" {
		fmt.Println()
		return nil
	}

	if selected, err = strconv.Atoi(response); err == nil && allowedOptions[selected] {
		return os.Options[selected-1].Command(data)
	} else {
		return fmt.Errorf("invalid option number entered")
	}
}

func (os OptionSelector) optionAllowedInRole(accessType auth.Role, enabledOptions []int, option int) bool {
	for _, o := range enabledOptions {
		if o == option {
			return os.OptionRoleFilter[accessType][option]
		}
	}
	return false
}