package lock

import (
	"path/filepath"
	"testing"
)

func TestExclusiveLock(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"vps-forge.lock",
	)

	first, err := Acquire(path)

	if err != nil {
		t.Fatal(err)
	}

	defer first.Close()

	if _, err := Acquire(path); err == nil {
		t.Fatal(
			"expected second lock acquisition to fail",
		)
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := Acquire(path)

	if err != nil {
		t.Fatalf(
			"expected lock to become available: %v",
			err,
		)
	}

	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}
