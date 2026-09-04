package accounts

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// execer is the test seam over command execution: tests replace it to
// capture argv and stdin without running real admin tools. It returns
// combined stdout+stderr.
var execer = realExec

func realExec(name string, args []string, stdin string) (string, error) {
	cmd := exec.Command(name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// run executes one admin tool and folds its output into the error.
func run(name string, args []string, stdin string) error {
	out, err := execer(name, args, stdin)
	if err == nil {
		return nil
	}
	if out != "" {
		return fmt.Errorf("%s: %s", name, out)
	}
	return fmt.Errorf("%s: %w", name, err)
}
