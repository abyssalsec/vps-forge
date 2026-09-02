package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"state",
		"state.json",
	)

	store := Store{
		Path: path,
	}

	expected := State{
		ToolVersion:  "0.2.0",
		ApplyID:      "test-apply",
		ConfigSHA256: "abc123",
		AppliedAt: time.Date(
			2026,
			9,
			2,
			12,
			0,
			0,
			0,
			time.UTC,
		),
		Modules: []string{
			"users",
			"ssh",
		},
		ManagedFiles: map[string]string{
			"/etc/test.conf": "deadbeef",
		},
	}

	if err := store.Save(
		expected,
	); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)

	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf(
			"expected mode 0600, got %o",
			info.Mode().Perm(),
		)
	}

	actual, exists, err := store.Load()

	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal(
			"expected state to exist",
		)
	}

	if actual.ApplyID !=
		expected.ApplyID {

		t.Fatalf(
			"expected apply ID %q, got %q",
			expected.ApplyID,
			actual.ApplyID,
		)
	}

	if actual.ManagedFiles["/etc/test.conf"] != "deadbeef" {

		t.Fatal(
			"managed file hash mismatch",
		)
	}
}

func TestHashFile(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"config.yaml",
	)

	if err := os.WriteFile(
		path,
		[]byte("vps-forge\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	first, err := HashFile(path)

	if err != nil {
		t.Fatal(err)
	}

	second, err := HashFile(path)

	if err != nil {
		t.Fatal(err)
	}

	if first == "" || first != second {
		t.Fatal(
			"expected deterministic SHA256 hash",
		)
	}
}
