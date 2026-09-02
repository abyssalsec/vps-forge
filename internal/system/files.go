package system

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
)

var applyIDPattern = regexp.MustCompile(
	`^[A-Za-z0-9._-]+$`,
)

type Validator func(
	context.Context,
	string,
) error

type FileSpec struct {
	Target   string
	Content  []byte
	Mode     fs.FileMode
	SetOwner bool
	UID      int
	GID      int
	Validate Validator
}

type ManagedFile struct {
	Target       string
	BackupPath   string
	HadOriginal  bool
	Changed      bool
	BeforeSHA256 string
	AfterSHA256  string
}

type FileManager struct {
	BackupRoot string
}

type fileMetadata struct {
	mode fs.FileMode
	uid  int
	gid  int
}

func hashBytes(
	data []byte,
) string {

	sum := sha256.Sum256(data)

	return hex.EncodeToString(
		sum[:],
	)
}

func metadata(
	info os.FileInfo,
) (fileMetadata, error) {

	stat, ok := info.Sys().(*syscall.Stat_t)

	if !ok {
		return fileMetadata{}, fmt.Errorf(
			"unsupported file metadata type",
		)
	}

	return fileMetadata{
		mode: info.Mode().Perm(),
		uid:  int(stat.Uid),
		gid:  int(stat.Gid),
	}, nil
}

func validateRegularTarget(
	path string,
) (os.FileInfo, bool, error) {

	info, err := os.Lstat(path)

	if os.IsNotExist(err) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return nil, false, fmt.Errorf(
			"managed file target must not be a symlink: %s",
			path,
		)
	}

	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf(
			"managed file target is not a regular file: %s",
			path,
		)
	}

	return info, true, nil
}

func syncDirectory(
	path string,
) {
	directory, err := os.Open(path)

	if err != nil {
		return
	}

	_ = directory.Sync()
	_ = directory.Close()
}

func writeTemporaryFile(
	parent string,
	content []byte,
	mode fs.FileMode,
	setOwner bool,
	uid int,
	gid int,
) (string, error) {

	temp, err := os.CreateTemp(
		parent,
		".vps-forge-*",
	)

	if err != nil {
		return "", err
	}

	path := temp.Name()
	keep := false

	defer func() {
		if !keep {
			_ = temp.Close()
			_ = os.Remove(path)
		}
	}()

	if _, err := temp.Write(
		content,
	); err != nil {
		return "", err
	}

	if err := temp.Chmod(
		mode.Perm(),
	); err != nil {
		return "", err
	}

	if setOwner {
		if err := temp.Chown(
			uid,
			gid,
		); err != nil {
			return "", err
		}
	}

	if err := temp.Sync(); err != nil {
		return "", err
	}

	if err := temp.Close(); err != nil {
		return "", err
	}

	keep = true
	return path, nil
}

func backupFile(
	source string,
	backup string,
	info os.FileInfo,
) error {

	if existing, err := os.Lstat(
		backup,
	); err == nil {

		if existing.Mode()&
			os.ModeSymlink != 0 {

			return fmt.Errorf(
				"backup path must not be a symlink",
			)
		}

		if !existing.Mode().IsRegular() {
			return fmt.Errorf(
				"backup path is not a regular file",
			)
		}

		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	meta, err := metadata(info)

	if err != nil {
		return err
	}

	if err := os.MkdirAll(
		filepath.Dir(backup),
		0o700,
	); err != nil {
		return err
	}

	sourceFile, err := os.Open(source)

	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destination, err := os.OpenFile(
		backup,
		os.O_CREATE|
			os.O_EXCL|
			os.O_WRONLY,
		0o600,
	)

	if err != nil {
		return err
	}

	success := false

	defer func() {
		_ = destination.Close()

		if !success {
			_ = os.Remove(backup)
		}
	}()

	if _, err := io.Copy(
		destination,
		sourceFile,
	); err != nil {
		return err
	}

	if err := destination.Chmod(
		meta.mode,
	); err != nil {
		return err
	}

	if err := destination.Chown(
		meta.uid,
		meta.gid,
	); err != nil {
		return err
	}

	if err := destination.Sync(); err != nil {
		return err
	}

	if err := destination.Close(); err != nil {
		return err
	}

	success = true
	return nil
}

func (m FileManager) Apply(
	ctx context.Context,
	applyID string,
	spec FileSpec,
) (ManagedFile, error) {

	if !applyIDPattern.MatchString(
		applyID,
	) {
		return ManagedFile{}, fmt.Errorf(
			"invalid apply ID",
		)
	}

	if m.BackupRoot == "" ||
		!filepath.IsAbs(m.BackupRoot) {

		return ManagedFile{}, fmt.Errorf(
			"backup root must be an absolute path",
		)
	}

	if spec.Target == "" ||
		!filepath.IsAbs(spec.Target) {

		return ManagedFile{}, fmt.Errorf(
			"managed file target must be absolute",
		)
	}

	if spec.Mode.Perm() == 0 {
		return ManagedFile{}, fmt.Errorf(
			"managed file mode cannot be zero",
		)
	}

	if spec.SetOwner &&
		(spec.UID < 0 || spec.GID < 0) {

		return ManagedFile{}, fmt.Errorf(
			"managed file owner IDs cannot be negative",
		)
	}

	if err := ctx.Err(); err != nil {
		return ManagedFile{}, err
	}

	parent := filepath.Dir(
		spec.Target,
	)

	parentInfo, err := os.Lstat(
		parent,
	)

	if err != nil {
		return ManagedFile{}, fmt.Errorf(
			"inspect managed file directory: %w",
			err,
		)
	}

	if parentInfo.Mode()&
		os.ModeSymlink != 0 {

		return ManagedFile{}, fmt.Errorf(
			"managed file parent must not be a symlink",
		)
	}

	if !parentInfo.IsDir() {
		return ManagedFile{}, fmt.Errorf(
			"managed file parent is not a directory",
		)
	}

	info, exists, err :=
		validateRegularTarget(
			spec.Target,
		)

	if err != nil {
		return ManagedFile{}, err
	}

	result := ManagedFile{
		Target:      spec.Target,
		HadOriginal: exists,
		AfterSHA256: hashBytes(
			spec.Content,
		),
	}

	if exists {
		currentContent, err := os.ReadFile(
			spec.Target,
		)

		if err != nil {
			return ManagedFile{}, fmt.Errorf(
				"read managed file: %w",
				err,
			)
		}

		result.BeforeSHA256 =
			hashBytes(currentContent)

		currentMetadata, err := metadata(
			info,
		)

		if err != nil {
			return ManagedFile{}, err
		}

		contentMatches :=
			result.BeforeSHA256 ==
				result.AfterSHA256

		modeMatches :=
			currentMetadata.mode ==
				spec.Mode.Perm()

		ownerMatches := true

		if spec.SetOwner {
			ownerMatches =
				currentMetadata.uid ==
					spec.UID &&
					currentMetadata.gid ==
						spec.GID
		}

		if contentMatches &&
			modeMatches &&
			ownerMatches {

			return result, nil
		}
	}

	tempPath, err := writeTemporaryFile(
		parent,
		spec.Content,
		spec.Mode,
		spec.SetOwner,
		spec.UID,
		spec.GID,
	)

	if err != nil {
		return ManagedFile{}, fmt.Errorf(
			"prepare managed file: %w",
			err,
		)
	}

	published := false

	defer func() {
		if !published {
			_ = os.Remove(tempPath)
		}
	}()

	if spec.Validate != nil {
		if err := spec.Validate(
			ctx,
			tempPath,
		); err != nil {

			return ManagedFile{}, fmt.Errorf(
				"managed file validation failed: %w",
				err,
			)
		}
	}

	if exists {
		relativeTarget := spec.Target[1:]

		backupPath := filepath.Join(
			m.BackupRoot,
			applyID,
			relativeTarget,
		)

		if err := backupFile(
			spec.Target,
			backupPath,
			info,
		); err != nil {
			return ManagedFile{}, fmt.Errorf(
				"backup managed file: %w",
				err,
			)
		}

		result.BackupPath =
			backupPath
	}

	if err := os.Rename(
		tempPath,
		spec.Target,
	); err != nil {
		return ManagedFile{}, fmt.Errorf(
			"publish managed file: %w",
			err,
		)
	}

	syncDirectory(parent)

	result.Changed = true
	published = true

	return result, nil
}

func (m FileManager) Restore(
	ctx context.Context,
	result ManagedFile,
) error {

	if !result.Changed {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	parent := filepath.Dir(
		result.Target,
	)

	if current, exists, err :=
		validateRegularTarget(
			result.Target,
		); err != nil {

		return err
	} else if exists &&
		current.Mode()&os.ModeSymlink != 0 {

		return fmt.Errorf(
			"refusing to restore over symlink",
		)
	}

	if !result.HadOriginal {
		if err := os.Remove(
			result.Target,
		); err != nil &&
			!os.IsNotExist(err) {

			return fmt.Errorf(
				"remove newly created managed file: %w",
				err,
			)
		}

		syncDirectory(parent)
		return nil
	}

	if result.BackupPath == "" {
		return fmt.Errorf(
			"managed file backup path is missing",
		)
	}

	backupInfo, exists, err :=
		validateRegularTarget(
			result.BackupPath,
		)

	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf(
			"managed file backup does not exist",
		)
	}

	meta, err := metadata(
		backupInfo,
	)

	if err != nil {
		return err
	}

	content, err := os.ReadFile(
		result.BackupPath,
	)

	if err != nil {
		return fmt.Errorf(
			"read managed file backup: %w",
			err,
		)
	}

	tempPath, err := writeTemporaryFile(
		parent,
		content,
		meta.mode,
		true,
		meta.uid,
		meta.gid,
	)

	if err != nil {
		return fmt.Errorf(
			"prepare managed file restore: %w",
			err,
		)
	}

	published := false

	defer func() {
		if !published {
			_ = os.Remove(tempPath)
		}
	}()

	if err := os.Rename(
		tempPath,
		result.Target,
	); err != nil {
		return fmt.Errorf(
			"restore managed file: %w",
			err,
		)
	}

	syncDirectory(parent)

	published = true
	return nil
}
