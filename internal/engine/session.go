package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/abyssalsec/vps-forge/internal/journal"
	"github.com/abyssalsec/vps-forge/internal/lock"
	"github.com/abyssalsec/vps-forge/internal/module"
	"github.com/abyssalsec/vps-forge/internal/state"
	"github.com/abyssalsec/vps-forge/internal/system"
)

type ApplyPaths struct {
	Lock       string
	State      string
	JournalDir string
	BackupRoot string
}

func DefaultApplyPaths() ApplyPaths {
	return ApplyPaths{
		Lock: "/run/lock/vps-forge.lock",

		State: "/var/lib/vps-forge/state.json",

		JournalDir: "/var/log/vps-forge",

		BackupRoot: "/var/lib/vps-forge/backups",
	}
}

type ApplySession struct {
	ID           string
	ConfigSHA256 string
	ToolVersion  string
	StartedAt    time.Time
	Paths        ApplyPaths

	lock    *lock.Lock
	journal *journal.Writer
}

func newApplyID() (string, error) {
	random := make(
		[]byte,
		6,
	)

	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf(
			"generate apply ID: %w",
			err,
		)
	}

	return time.Now().
			UTC().
			Format("20060102T150405Z") +
			"-" +
			hex.EncodeToString(random),
		nil
}

func BeginApply(
	configPath string,
	toolVersion string,
	paths ApplyPaths,
) (*ApplySession, error) {

	if configPath == "" {
		return nil, fmt.Errorf(
			"configuration path cannot be empty",
		)
	}

	if toolVersion == "" {
		return nil, fmt.Errorf(
			"tool version cannot be empty",
		)
	}

	id, err := newApplyID()

	if err != nil {
		return nil, err
	}

	applyLock, err := lock.Acquire(
		paths.Lock,
	)

	if err != nil {
		return nil, err
	}

	configHash, err := state.HashFile(
		configPath,
	)

	if err != nil {
		_ = applyLock.Close()
		return nil, err
	}

	journalPath := filepath.Join(
		paths.JournalDir,
		"apply-"+id+".jsonl",
	)

	writer, err := journal.Open(
		journalPath,
		id,
	)

	if err != nil {
		_ = applyLock.Close()
		return nil, err
	}

	session := &ApplySession{
		ID:           id,
		ConfigSHA256: configHash,
		ToolVersion:  toolVersion,
		StartedAt:    time.Now().UTC(),
		Paths:        paths,
		lock:         applyLock,
		journal:      writer,
	}

	if err := writer.Record(
		journal.Entry{
			Event:  "apply",
			Status: "started",
		},
	); err != nil {

		_ = writer.Close()
		_ = applyLock.Close()

		return nil, err
	}

	return session, nil
}

func (s *ApplySession) Bind(
	environment module.Environment,
) module.Environment {

	environment.ApplyID = s.ID

	environment.Journal =
		s.journal

	environment.Files =
		system.FileManager{
			BackupRoot: s.Paths.BackupRoot,
		}

	return environment
}

func (s *ApplySession) Commit(
	modules []string,
	managedFiles []system.ManagedFile,
) error {

	managed := make(
		map[string]string,
	)

	for _, file := range managedFiles {
		if file.Target == "" {
			continue
		}

		managed[file.Target] =
			file.AfterSHA256
	}

	moduleNames := append(
		[]string(nil),
		modules...,
	)

	sort.Strings(moduleNames)

	store := state.Store{
		Path: s.Paths.State,
	}

	if err := store.Save(
		state.State{
			ToolVersion:  s.ToolVersion,
			ApplyID:      s.ID,
			ConfigSHA256: s.ConfigSHA256,
			AppliedAt:    time.Now().UTC(),
			Modules:      moduleNames,
			ManagedFiles: managed,
		},
	); err != nil {
		return fmt.Errorf(
			"save apply state: %w",
			err,
		)
	}

	if err := s.journal.Record(
		journal.Entry{
			Event:  "apply",
			Status: "completed",
		},
	); err != nil {
		return fmt.Errorf(
			"record apply completion: %w",
			err,
		)
	}

	return nil
}

func (s *ApplySession) Fail(
	message string,
) error {

	return s.journal.Record(
		journal.Entry{
			Event:   "apply",
			Status:  "failed",
			Message: message,
		},
	)
}

func (s *ApplySession) Close() error {
	var journalErr error
	var lockErr error

	if s.journal != nil {
		journalErr =
			s.journal.Close()

		s.journal = nil
	}

	if s.lock != nil {
		lockErr = s.lock.Close()
		s.lock = nil
	}

	if journalErr != nil {
		return journalErr
	}

	if lockErr != nil {
		return lockErr
	}

	return nil
}
