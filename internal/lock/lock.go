package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Lock struct {
	file *os.File
	path string
}

func Acquire(path string) (*Lock, error) {
	if path == "" {
		return nil, fmt.Errorf("lock path cannot be empty")
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf(
			"lock path must be absolute: %q",
			path,
		)
	}

	if err := os.MkdirAll(
		filepath.Dir(path),
		0o755,
	); err != nil {
		return nil, fmt.Errorf(
			"create lock directory: %w",
			err,
		)
	}

	fd, err := unix.Open(
		path,
		unix.O_CREAT|
			unix.O_RDWR|
			unix.O_CLOEXEC|
			unix.O_NOFOLLOW,
		0o600,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"open apply lock: %w",
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
			"create apply lock file handle",
		)
	}

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()

		return nil, fmt.Errorf(
			"set apply lock permissions: %w",
			err,
		)
	}

	err = unix.Flock(
		int(file.Fd()),
		unix.LOCK_EX|unix.LOCK_NB,
	)

	if err != nil {
		_ = file.Close()

		if errors.Is(err, unix.EWOULDBLOCK) ||
			errors.Is(err, unix.EAGAIN) {

			return nil, fmt.Errorf(
				"another VPS Forge apply is already running",
			)
		}

		return nil, fmt.Errorf(
			"acquire apply lock: %w",
			err,
		)
	}

	if err := file.Truncate(0); err != nil {
		_ = unix.Flock(
			int(file.Fd()),
			unix.LOCK_UN,
		)

		_ = file.Close()

		return nil, fmt.Errorf(
			"truncate apply lock: %w",
			err,
		)
	}

	if _, err := file.Seek(0, 0); err != nil {
		_ = unix.Flock(
			int(file.Fd()),
			unix.LOCK_UN,
		)

		_ = file.Close()

		return nil, fmt.Errorf(
			"seek apply lock: %w",
			err,
		)
	}

	if _, err := fmt.Fprintf(
		file,
		"%d\n",
		os.Getpid(),
	); err != nil {
		_ = unix.Flock(
			int(file.Fd()),
			unix.LOCK_UN,
		)

		_ = file.Close()

		return nil, fmt.Errorf(
			"write apply lock: %w",
			err,
		)
	}

	if err := file.Sync(); err != nil {
		_ = unix.Flock(
			int(file.Fd()),
			unix.LOCK_UN,
		)

		_ = file.Close()

		return nil, fmt.Errorf(
			"sync apply lock: %w",
			err,
		)
	}

	return &Lock{
		file: file,
		path: path,
	}, nil
}

func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	unlockErr := unix.Flock(
		int(l.file.Fd()),
		unix.LOCK_UN,
	)

	closeErr := l.file.Close()
	l.file = nil

	if unlockErr != nil {
		return fmt.Errorf(
			"release apply lock: %w",
			unlockErr,
		)
	}

	if closeErr != nil {
		return fmt.Errorf(
			"close apply lock: %w",
			closeErr,
		)
	}

	return nil
}
