package click

import (
	"fmt"
	"strings"
)

const (
	evalTed = "%eval"
	khanTed = "%khan-eval"
	markOut = "%noun"
	markIn  = "%ted-eval"
	command = "%fyrd"
	desk    = "%base"
)

var (
	rid = 0
)

// FormatCommand formats the input command and its dependencies
func Click(args []string, useKhanThread bool) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	thread := evalTed
	if useKhanThread {
		thread = khanTed
	}

	var tedIn string
	if len(args) == 1 {
		tedIn = fmt.Sprintf("['%s']", args[0])
	} else {
		dependencies := strings.Join(args[1:], " ")
		tedIn = fmt.Sprintf("['%s' ['%s' [%s ~]]]", markIn, args[0], dependencies)
	}

	return fmt.Sprintf("[%d %s [%s %s %s %s]]", rid, command, desk, thread, markOut, tedIn), nil
}

// Example usage
func main() {
	args := []string{"your_command", "dependency1", "dependency2"}
	useKhanThread := true // Set to false if you don't want to use the Khan thread

	result, err := Click(args, useKhanThread)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(result)
}
