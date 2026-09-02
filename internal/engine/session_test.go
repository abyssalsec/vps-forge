package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abyssalsec/vps-forge/internal/module"
	"github.com/abyssalsec/vps-forge/internal/state"
	"github.com/abyssalsec/vps-forge/internal/system"
)

func TestApplySessionCommit(
	t *testing.T,
) {
	root := t.TempDir()

	configPath := filepath.Join(
		root,
		"forge.yaml",
	)

	if err := os.WriteFile(
		configPath,
		[]byte(
			"version: 1\nprofile: minimal\n",
		),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	paths := ApplyPaths{
		Lock: filepath.Join(
			root,
			"run",
			"vps-forge.lock",
		),

		State: filepath.Join(
			root,
			"state",
			"state.json",
		),

		JournalDir: filepath.Join(
			root,
			"log",
		),

		BackupRoot: filepath.Join(
			root,
			"backups",
		),
	}

	session, err := BeginApply(
		configPath,
		"0.2.0",
		paths,
	)

	if err != nil {
		t.Fatal(err)
	}

	defer session.Close()

	environment := session.Bind(
		module.Environment{},
	)

	if environment.ApplyID == "" {
		t.Fatal(
			"expected bound apply ID",
		)
	}

	if environment.Journal == nil {
		t.Fatal(
			"expected bound journal",
		)
	}

	if environment.Files.BackupRoot !=
		paths.BackupRoot {

		t.Fatal(
			"expected bound file manager",
		)
	}

	if err := session.Commit(
		[]string{
			"ssh",
			"users",
		},
		[]system.ManagedFile{
			{
				Target:      "/etc/test.conf",
				Changed:     true,
				AfterSHA256: "deadbeef",
			},
		},
	); err != nil {
		t.Fatal(err)
	}

	store := state.Store{
		Path: paths.State,
	}

	current, exists, err :=
		store.Load()

	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal(
			"expected committed state",
		)
	}

	if current.ToolVersion !=
		"0.2.0" {

		t.Fatalf(
			"unexpected tool version %q",
			current.ToolVersion,
		)
	}

	if current.ManagedFiles["/etc/test.conf"] != "deadbeef" {

		t.Fatal(
			"managed file was not committed",
		)
	}

	journalFiles, err := filepath.Glob(
		filepath.Join(
			paths.JournalDir,
			"apply-*.jsonl",
		),
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(journalFiles) != 1 {
		t.Fatalf(
			"expected one journal, got %d",
			len(journalFiles),
		)
	}
}
