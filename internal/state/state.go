package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const SchemaVersion = 1

type State struct {
	SchemaVersion int               `json:"schema_version"`
	ToolVersion   string            `json:"tool_version"`
	ApplyID       string            `json:"apply_id"`
	ConfigSHA256  string            `json:"config_sha256"`
	AppliedAt     time.Time         `json:"applied_at"`
	Modules       []string          `json:"modules"`
	ManagedFiles  map[string]string `json:"managed_files"`
}

type Store struct {
	Path string
}

func HashFile(path string) (string, error) {
	file, err := os.Open(path)

	if err != nil {
		return "", fmt.Errorf(
			"open file for hashing: %w",
			err,
		)
	}
	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(
		hash,
		file,
	); err != nil {
		return "", fmt.Errorf(
			"hash file: %w",
			err,
		)
	}

	return hex.EncodeToString(
		hash.Sum(nil),
	), nil
}

func (s Store) Load() (
	State,
	bool,
	error,
) {

	if s.Path == "" {
		return State{}, false, fmt.Errorf(
			"state path cannot be empty",
		)
	}

	info, err := os.Lstat(s.Path)

	if os.IsNotExist(err) {
		return State{
			SchemaVersion: SchemaVersion,
			ManagedFiles:  map[string]string{},
		}, false, nil
	}

	if err != nil {
		return State{}, false, fmt.Errorf(
			"stat state file: %w",
			err,
		)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return State{}, false, fmt.Errorf(
			"state path must not be a symlink",
		)
	}

	if !info.Mode().IsRegular() {
		return State{}, false, fmt.Errorf(
			"state path is not a regular file",
		)
	}

	file, err := os.Open(s.Path)

	if err != nil {
		return State{}, false, fmt.Errorf(
			"open state file: %w",
			err,
		)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var current State

	if err := decoder.Decode(
		&current,
	); err != nil {
		return State{}, false, fmt.Errorf(
			"decode state file: %w",
			err,
		)
	}

	if current.SchemaVersion !=
		SchemaVersion {

		return State{}, false, fmt.Errorf(
			"unsupported state schema version %d",
			current.SchemaVersion,
		)
	}

	if current.ManagedFiles == nil {
		current.ManagedFiles =
			map[string]string{}
	}

	return current, true, nil
}

func (s Store) Save(
	current State,
) error {

	if s.Path == "" {
		return fmt.Errorf(
			"state path cannot be empty",
		)
	}

	if !filepath.IsAbs(s.Path) {
		return fmt.Errorf(
			"state path must be absolute",
		)
	}

	if info, err := os.Lstat(s.Path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf(
				"state path must not be a symlink",
			)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf(
			"inspect state path: %w",
			err,
		)
	}

	directory := filepath.Dir(
		s.Path,
	)

	if err := os.MkdirAll(
		directory,
		0o700,
	); err != nil {
		return fmt.Errorf(
			"create state directory: %w",
			err,
		)
	}

	current.SchemaVersion =
		SchemaVersion

	if current.ManagedFiles == nil {
		current.ManagedFiles =
			map[string]string{}
	}

	temp, err := os.CreateTemp(
		directory,
		".vps-forge-state-*",
	)

	if err != nil {
		return fmt.Errorf(
			"create temporary state file: %w",
			err,
		)
	}

	tempPath := temp.Name()
	published := false

	defer func() {
		if !published {
			_ = temp.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf(
			"set state permissions: %w",
			err,
		)
	}

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(
		current,
	); err != nil {
		return fmt.Errorf(
			"encode state: %w",
			err,
		)
	}

	if err := temp.Sync(); err != nil {
		return fmt.Errorf(
			"sync state: %w",
			err,
		)
	}

	if err := temp.Close(); err != nil {
		return fmt.Errorf(
			"close state: %w",
			err,
		)
	}

	if err := os.Rename(
		tempPath,
		s.Path,
	); err != nil {
		return fmt.Errorf(
			"publish state: %w",
			err,
		)
	}

	directoryHandle, err := os.Open(
		directory,
	)

	if err == nil {
		_ = directoryHandle.Sync()
		_ = directoryHandle.Close()
	}

	published = true
	return nil
}
