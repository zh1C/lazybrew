package brew

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes brew commands and captures output.
type Runner struct {
	brewPath string
}

// NewRunner creates a new brew command runner.
func NewRunner() *Runner {
	brewPath, err := exec.LookPath("brew")
	if err != nil {
		brewPath = "/opt/homebrew/bin/brew"
	}
	return &Runner{brewPath: brewPath}
}

// CommandResult holds the result of a brew command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Run executes a brew command with the given arguments.
func (r *Runner) Run(args ...string) CommandResult {
	cmd := exec.Command(r.brewPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	return CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// RunWithCallback executes a brew command and sends live output to a callback.
func (r *Runner) RunWithCallback(onOutput func(line string), args ...string) CommandResult {
	cmd := exec.Command(r.brewPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CommandResult{Err: fmt.Errorf("failed to create stdout pipe: %w", err)}
	}

	if err := cmd.Start(); err != nil {
		return CommandResult{Err: fmt.Errorf("failed to start command: %w", err)}
	}

	var outputBuf bytes.Buffer
	buf := make([]byte, 1024)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			outputBuf.WriteString(chunk)
			lines := strings.Split(chunk, "\n")
			for _, line := range lines {
				if line != "" {
					onOutput(line)
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	err = cmd.Wait()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	return CommandResult{
		Stdout:   outputBuf.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// CommandString returns the full command string for display purposes.
func (r *Runner) CommandString(args ...string) string {
	return "brew " + strings.Join(args, " ")
}
