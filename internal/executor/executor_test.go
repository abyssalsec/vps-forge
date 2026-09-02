package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecutorRunsCommand(t *testing.T) {
	current := OSExecutor{}

	result, err := current.Run(
		context.Background(),
		Command{
			Name: "printf",
			Args: []string{
				"%s",
				"vps-forge",
			},
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if result.Stdout != "vps-forge" {
		t.Fatalf(
			"unexpected stdout %q",
			result.Stdout,
		)
	}
}

func TestExecutorDoesNotInterpretShellSyntax(
	t *testing.T,
) {
	current := OSExecutor{}

	value := "$(id); echo injected"

	result, err := current.Run(
		context.Background(),
		Command{
			Name: "printf",
			Args: []string{
				"%s",
				value,
			},
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if result.Stdout != value {
		t.Fatalf(
			"shell syntax was altered: %q",
			result.Stdout,
		)
	}
}

func TestExecutorTimeout(t *testing.T) {
	current := OSExecutor{}

	_, err := current.Run(
		context.Background(),
		Command{
			Name:    "sleep",
			Args:    []string{"2"},
			Timeout: 20 * time.Millisecond,
		},
	)

	if err == nil {
		t.Fatal(
			"expected timeout error",
		)
	}

	if !strings.Contains(
		err.Error(),
		"exceeded timeout",
	) {
		t.Fatalf(
			"unexpected timeout error: %v",
			err,
		)
	}
}
