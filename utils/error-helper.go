package utils

import (
	"fmt"
	"os"
)

func PrintExitIfError(e error, prefix string, exitCode int) {
	if e != nil {
		if len(prefix) != 0 {
			fmt.Fprintf(os.Stderr, "%s : %s\r\n", prefix, e.Error())
		} else {
			fmt.Fprintf(os.Stderr, "%s\r\n", e.Error())
		}
		os.Exit(exitCode)
	}

}
