package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DefaultPATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

type Command struct {
	Name    string
	Args    []string
	Env     map[string]string
	Timeout time.Duration
}

type Result struct {
	Executable string
	Stdout     string
	Stderr     string
	ExitCode   int
}

type Executor interface {
	Run(context.Context, Command) (Result, error)
}

type OSExecutor struct {
	DefaultTimeout time.Duration
}

type CommandError struct {
	Executable string
	Args       []string
	ExitCode   int
	Stderr     string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf(
		"command %s failed with exit code %d",
		e.Executable,
		e.ExitCode,
	)
}

func resolveExecutable(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty executable name")
	}

	if strings.ContainsRune(name, filepath.Separator) {
		if !filepath.IsAbs(name) {
			return "", fmt.Errorf(
				"executable path must be absolute: %q",
				name,
			)
		}

		info, err := os.Stat(name)
		if err != nil {
			return "", fmt.Errorf(
				"stat executable %q: %w",
				name,
				err,
			)
		}

		if !info.Mode().IsRegular() ||
			info.Mode().Perm()&0o111 == 0 {

			return "", fmt.Errorf(
				"path is not executable: %q",
				name,
			)
		}

		return name, nil
	}

	for _, directory := range strings.Split(
		DefaultPATH,
		":",
	) {
		candidate := filepath.Join(
			directory,
			name,
		)

		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}

		if info.Mode().IsRegular() &&
			info.Mode().Perm()&0o111 != 0 {

			return candidate, nil
		}
	}

	return "", fmt.Errorf(
		"executable %q was not found in trusted PATH",
		name,
	)
}

func buildEnvironment(
	extra map[string]string,
) ([]string, error) {

	environment := []string{
		"PATH=" + DefaultPATH,
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
	}

	keys := make([]string, 0, len(extra))

	for key := range extra {
		if key == "" ||
			strings.Contains(key, "=") ||
			strings.ContainsRune(key, '\x00') {

			return nil, fmt.Errorf(
				"invalid environment variable name %q",
				key,
			)
		}

		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		value := extra[key]

		if strings.ContainsRune(value, '\x00') {
			return nil, fmt.Errorf(
				"environment variable %q contains NUL",
				key,
			)
		}

		environment = append(
			environment,
			key+"="+value,
		)
	}

	return environment, nil
}

func (e OSExecutor) Run(
	ctx context.Context,
	command Command,
) (Result, error) {

	executable, err := resolveExecutable(
		command.Name,
	)

	if err != nil {
		return Result{}, err
	}

	timeout := command.Timeout

	if timeout <= 0 {
		timeout = e.DefaultTimeout
	}

	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	commandContext, cancel := context.WithTimeout(
		ctx,
		timeout,
	)
	defer cancel()

	environment, err := buildEnvironment(
		command.Env,
	)

	if err != nil {
		return Result{}, err
	}

	process := exec.CommandContext(
		commandContext,
		executable,
		command.Args...,
	)

	process.Env = environment

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	process.Stdout = &stdout
	process.Stderr = &stderr

	err = process.Run()

	result := Result{
		Executable: executable,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   0,
	}

	if commandContext.Err() ==
		context.DeadlineExceeded {

		return result, fmt.Errorf(
			"command %s exceeded timeout %s",
			executable,
			timeout,
		)
	}

	if err == nil {
		return result, nil
	}

	var exitError *exec.ExitError

	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()

		return result, &CommandError{
			Executable: executable,
			Args:       append([]string(nil), command.Args...),
			ExitCode:   result.ExitCode,
			Stderr:     result.Stderr,
		}
	}

	return result, fmt.Errorf(
		"execute %s: %w",
		executable,
		err,
	)
}
