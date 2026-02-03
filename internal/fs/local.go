package fs

import (
	"os"
	"path/filepath"
)

// LocalFS implements FileSystem using the local filesystem.
type LocalFS struct {
	root string
}

// NewLocalFS creates a LocalFS rooted at the given directory.
func NewLocalFS(root string) *LocalFS {
	return &LocalFS{root: root}
}

func (l *LocalFS) abs(path string) string {
	if path == "" || path == "." {
		return l.root
	}
	return filepath.Join(l.root, path)
}

// ReadFile reads the contents of the file at the given path relative to the root.
func (l *LocalFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(l.abs(path))
}

// Stat returns metadata for the file or directory at the given path relative to the root.
func (l *LocalFS) Stat(path string) (FileInfo, error) {
	info, err := os.Stat(l.abs(path))
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    info.Name(),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// ReadDir lists the immediate children of the directory at the given path relative to the root.
func (l *LocalFS) ReadDir(path string) ([]DirEntry, error) {
	entries, err := os.ReadDir(l.abs(path))
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		}
	}
	return result, nil
}
