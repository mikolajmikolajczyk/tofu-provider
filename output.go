package main

import (
	"fmt"
	"os"
)

func logOK(msg string) {
	fmt.Printf("\033[32m✓\033[0m  %s\n", msg)
}

func logInfo(msg string) {
	fmt.Printf("\033[36minfo:\033[0m %s\n", msg)
}

func logError(msg string) {
	fmt.Fprintf(os.Stderr, "\033[31merror:\033[0m %s\n", msg)
}
