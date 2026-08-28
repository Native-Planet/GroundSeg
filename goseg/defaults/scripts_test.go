package defaults

import (
	"strings"
	"testing"
)

func TestRollChopScriptCommandSequence(t *testing.T) {
	commands := []string{
		"urbit -Lx $ttyflag --loom $loom $dirname || true",
		"urbit roll --loom $loom $dirname || true",
		"urbit -Lx $ttyflag --loom $loom $dirname || true",
		"urbit roll --loom $loom $dirname || true",
		"urbit chop --loom $loom $dirname",
	}

	remaining := RollChopScript
	for _, command := range commands {
		index := strings.Index(remaining, command)
		if index == -1 {
			t.Fatalf("roll/chop script is missing command %q in the expected order", command)
		}
		remaining = remaining[index+len(command):]
	}
}
