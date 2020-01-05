package utils

import (
	"fmt"
	"os"

	"github.com/mevansam/goutils/term"
	"github.com/mevansam/goutils/utils"
)

func ShowErrorAndExit(message string) {

	var (
		format string
	)

	if message[len(message)-1] == '.' {
		format = term.RED + "\nError! %s\n" + term.NC
	} else {
		format = term.RED + "\nError! %s.\n" + term.NC
	}

	fmt.Println(utils.FormatMessage(7, 80, false, true, format, message))
	os.Exit(1)
}
