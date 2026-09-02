package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	ApplyID   string    `json:"apply_id"`
	Event     string    `json:"event"`
	Module    string    `json:"module,omitempty"`
	ChangeID  string    `json:"change_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	Message   string    `json:"message,omitempty"`
}

type Recorder interface {
	Record(Entry) error
}

type Writer struct {
	mu      sync.Mutex
	file    *os.File
	applyID string
}

func Open(
	path string,
	applyID string,
) (*Writer, error) {

	if path == "" {
		return nil, fmt.Errorf(
			"journal path cannot be empty",
		)
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf(
			"journal path must be absolute",
		)
	}

	if applyID == "" {
		return nil, fmt.Errorf(
			"journal apply ID cannot be empty",
		)
	}

	if err := os.MkdirAll(
		filepath.Dir(path),
		0o700,
	); err != nil {
		return nil, fmt.Errorf(
			"create journal directory: %w",
			err,
		)
	}

	fd, err := unix.Open(
		path,
		unix.O_CREAT|
			unix.O_WRONLY|
			unix.O_APPEND|
			unix.O_CLOEXEC|
			unix.O_NOFOLLOW,
		0o600,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"open journal: %w",
			err,
		)
	}

	file := os.NewFile(
		uintptr(fd),
		path,
	)

	if file == nil {
		_ = unix.Close(fd)

		return nil, fmt.Errorf(
			"create journal file handle",
		)
	}

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()

		return nil, fmt.Errorf(
			"set journal permissions: %w",
			err,
		)
	}

	return &Writer{
		file:    file,
		applyID: applyID,
	}, nil
}

func (w *Writer) Record(
	entry Entry,
) error {

	if w == nil || w.file == nil {
		return fmt.Errorf(
			"journal is not open",
		)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	if entry.ApplyID == "" {
		entry.ApplyID = w.applyID
	}

	encoder := json.NewEncoder(
		w.file,
	)

	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf(
			"write journal event: %w",
			err,
		)
	}

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf(
			"sync journal: %w",
			err,
		)
	}

	return nil
}

func (w *Writer) Close() error {
	if w == nil || w.file == nil {
		return nil
	}

	err := w.file.Close()
	w.file = nil

	if err != nil {
		return fmt.Errorf(
			"close journal: %w",
			err,
		)
	}

	return nil
}
