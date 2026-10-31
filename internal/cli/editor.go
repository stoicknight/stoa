package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runEditor starts the configured editor on one file and waits for it.
// In non-interactive contexts (pipes, cron, subshells) the editor is skipped:
// editors need a tty, and anything else just blocks.
func runEditor(editor, file string) error {
	if !ttyInteractive() {
		fmt.Printf("(skipping editor: not an interactive tty; edit %s manually)\n", file)
		return nil
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return failf("no editor configured")
	}
	args := append(parts[1:], file)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return failf("editor %s failed: %v", parts[0], err)
	}
	return nil
}
