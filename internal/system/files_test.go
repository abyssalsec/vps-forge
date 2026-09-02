package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestManagedFileApplyAndRestore(
	t *testing.T,
) {
	root := t.TempDir()

	target := filepath.Join(
		root,
		"service.conf",
	)

	backupRoot := filepath.Join(
		root,
		"backups",
	)

	if err := os.WriteFile(
		target,
		[]byte("old=true\n"),
		0o640,
	); err != nil {
		t.Fatal(err)
	}

	manager := FileManager{
		BackupRoot: backupRoot,
	}

	result, err := manager.Apply(
		context.Background(),
		"apply-test",
		FileSpec{
			Target:  target,
			Content: []byte("new=true\n"),
			Mode:    0o640,

			Validate: func(
				_ context.Context,
				path string,
			) error {

				content, err := os.ReadFile(
					path,
				)

				if err != nil {
					return err
				}

				if string(content) !=
					"new=true\n" {

					return fmt.Errorf(
						"unexpected content",
					)
				}

				return nil
			},
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if !result.Changed {
		t.Fatal(
			"expected managed file to change",
		)
	}

	if result.BackupPath == "" {
		t.Fatal(
			"expected original backup",
		)
	}

	content, err := os.ReadFile(target)

	if err != nil {
		t.Fatal(err)
	}

	if string(content) !=
		"new=true\n" {

		t.Fatalf(
			"unexpected managed content %q",
			content,
		)
	}

	second, err := manager.Apply(
		context.Background(),
		"apply-test",
		FileSpec{
			Target:  target,
			Content: []byte("new=true\n"),
			Mode:    0o640,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	if second.Changed {
		t.Fatal(
			"expected second apply to be idempotent",
		)
	}

	if err := manager.Restore(
		context.Background(),
		result,
	); err != nil {
		t.Fatal(err)
	}

	content, err = os.ReadFile(target)

	if err != nil {
		t.Fatal(err)
	}

	if string(content) !=
		"old=true\n" {

		t.Fatalf(
			"restore produced %q",
			content,
		)
	}
}

func TestValidationFailureLeavesTargetUntouched(
	t *testing.T,
) {
	root := t.TempDir()

	target := filepath.Join(
		root,
		"service.conf",
	)

	if err := os.WriteFile(
		target,
		[]byte("safe\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	manager := FileManager{
		BackupRoot: filepath.Join(
			root,
			"backups",
		),
	}

	_, err := manager.Apply(
		context.Background(),
		"apply-test",
		FileSpec{
			Target:  target,
			Content: []byte("broken\n"),
			Mode:    0o600,

			Validate: func(
				context.Context,
				string,
			) error {

				return fmt.Errorf(
					"configuration rejected",
				)
			},
		},
	)

	if err == nil {
		t.Fatal(
			"expected validation failure",
		)
	}

	content, readErr :=
		os.ReadFile(target)

	if readErr != nil {
		t.Fatal(readErr)
	}

	if string(content) != "safe\n" {
		t.Fatalf(
			"validation failure modified target: %q",
			content,
		)
	}
}
