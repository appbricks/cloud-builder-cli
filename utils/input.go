package utils

import (
	"fmt"
	"os"

	"github.com/mevansam/goutils/term"
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
			fmt.Println(
				term.RED +
					"\nInput aborted.\n" +
					term.NC,
			)
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
			fmt.Println(
				term.RED +
					"\nInput aborted.\n" +
					term.NC,
			)
			os.Exit(1)
		} else {
			ShowErrorAndExit(err.Error())
		}
	}
	return response
}
