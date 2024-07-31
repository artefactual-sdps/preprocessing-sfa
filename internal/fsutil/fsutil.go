package fsutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileExists returns true if path references an existing file.  FileExists will
// return false if path doesn't reference an existing file, or if
// `os.Stat(path)` returns an error (e.g. ErrPermission).
func FileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// FindFilename searches root for files named name and returns the paths of any
// matching files.
func FindFilename(root, name string) (found []string, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Name() == name {
			found = append(found, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find filename: %v", err)
	}

	return found, nil
}
