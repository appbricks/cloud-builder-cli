package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gookit/color"
	"github.com/peterh/liner"
)

func GetUserInput(
	prompt string,
) string {

	var (
		err error

		response string
	)

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	defer func() {
		line.Close()
	}()

	if response, err = line.Prompt(
		prompt,
	); err != nil {

		if err == liner.ErrPromptAborted {
			fmt.Println(color.Red.Render("\nInput aborted.\n"))
			os.Exit(1)
		} else {
			ShowErrorAndExit(err.Error())
		}
	}
	return response
}

func GetUserInputFromList(
	prompt string,
	selected string,
	options []string,
	validate bool,
) string {

	var (
		err error

		response string
	)

	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) []string {
		return options
	})
	defer func() {
		line.Close()
	}()

	if response, err = line.PromptWithSuggestion(
		prompt, selected, -1,
	); err != nil {

		if err == liner.ErrPromptAborted {
			fmt.Println(color.Red.Render("\nInput aborted.\n"))
			os.Exit(1)
		} else {
			ShowErrorAndExit(err.Error())
		}
	}
	if validate {
		for _, o := range options {
			if response == o {
				return response
			}
		}
	
		fmt.Println(
			color.Red.Render(
				fmt.Sprintf("\nInvalid selection '%s'. Selection must be one of the following:\n", response),
			),
		)
		for _, o := range options {
			fmt.Println(
				color.Red.Render(
					fmt.Sprintf("- %s", o),
				),
			)
		}
		fmt.Println()
		os.Exit(1)	
	}
	return response
}

func GetYesNoUserInput(prompt string, defaultRespone bool) (bool, error) {

	var(
		err error

		defaultInput,
		input string
	)
	
	line := liner.NewLiner()
	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) []string {
		return []string{"yes", "no"}
	})
	defer func() {
		line.Close()
	}()

	if defaultRespone {
		defaultInput = "yes"
	} else {
		defaultInput = "no"
	}

	if input, err = line.PromptWithSuggestion(prompt, defaultInput, -1); err != nil {
		return defaultRespone, err
	}
	line.SetCompleter(nil)

	input = strings.ToLower(input)
	if match, err := regexp.Match(`^((y(es)?)|(no?))$`, []byte(input)); !match || err != nil {
		return defaultRespone, fmt.Errorf("invalid input.")
	}
	return input == "yes" || input == "y", nil
}
