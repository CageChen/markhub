// Package fs provides filesystem abstractions for reading files from local disk or git repos.
package fs

import "time"

// FileInfo holds file metadata.
type FileInfo struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// DirEntry represents a single directory entry.
type DirEntry struct {
	Name  string
	IsDir bool
}

// FileSystem abstracts file operations so callers can work with either
// the local filesystem or a git object database.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (FileInfo, error)
	ReadDir(path string) ([]DirEntry, error)
}
