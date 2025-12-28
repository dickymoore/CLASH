package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm asks the user to proceed. Default is no.
func Confirm(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

// RequirePhrase prompts for an exact phrase.
func RequirePhrase(prompt, phrase string) bool {
	fmt.Printf("%s\nType \"%s\" to continue: ", prompt, phrase)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.TrimSpace(line) == phrase
}
