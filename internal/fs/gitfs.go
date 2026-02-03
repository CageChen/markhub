package fs

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GitFS implements FileSystem by reading from a git ref (branch, tag, or commit).
type GitFS struct {
	repoPath string
	ref      string
}

// NewGitFS creates a GitFS that reads files from the given ref in the repository at repoPath.
func NewGitFS(repoPath, ref string) *GitFS {
	return &GitFS{repoPath: repoPath, ref: ref}
}

func (g *GitFS) git(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", g.repoPath}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// ReadFile reads the contents of the file at the given path from the git ref.
func (g *GitFS) ReadFile(path string) ([]byte, error) {
	objPath := path
	if objPath == "" || objPath == "." {
		return nil, fmt.Errorf("cannot read directory as file")
	}
	cmd := exec.Command("git", "-C", g.repoPath, "show", g.ref+":"+objPath)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if strings.Contains(stderr, "does not exist") || strings.Contains(stderr, "not exist") {
				return nil, os.ErrNotExist
			}
			return nil, fmt.Errorf("git show: %s", stderr)
		}
		return nil, err
	}
	return out, nil
}

// Stat returns metadata for the file or directory at the given path in the git ref.
func (g *GitFS) Stat(path string) (FileInfo, error) {
	objPath := path
	if objPath == "" {
		objPath = "."
	}

	// For root, check if the ref exists at all
	if objPath == "." {
		_, err := g.git("rev-parse", "--verify", g.ref)
		if err != nil {
			return FileInfo{}, os.ErrNotExist
		}
		modTime := g.getModTime(".")
		return FileInfo{
			Name:    g.ref,
			IsDir:   true,
			ModTime: modTime,
		}, nil
	}

	// Use ls-tree to determine if the path is a file or directory
	out, err := g.git("ls-tree", g.ref, objPath)
	if err != nil {
		return FileInfo{}, os.ErrNotExist
	}

	out = strings.TrimSpace(out)
	if out == "" {
		// Maybe it's a directory — try with trailing slash
		out, err = g.git("ls-tree", g.ref, objPath+"/")
		if err != nil || strings.TrimSpace(out) == "" {
			return FileInfo{}, os.ErrNotExist
		}
		// It's a directory
		modTime := g.getModTime(objPath)
		return FileInfo{
			Name:    baseName(objPath),
			IsDir:   true,
			ModTime: modTime,
		}, nil
	}

	// Parse ls-tree output: "<mode> <type> <hash>\t<name>"
	fields := strings.Fields(out)
	if len(fields) < 4 {
		return FileInfo{}, os.ErrNotExist
	}
	objType := fields[1]

	modTime := g.getModTime(objPath)

	if objType == "tree" {
		return FileInfo{
			Name:    baseName(objPath),
			IsDir:   true,
			ModTime: modTime,
		}, nil
	}

	// It's a blob — get its size
	var size int64
	sizeOut, err := g.git("cat-file", "-s", g.ref+":"+objPath)
	if err == nil {
		size, _ = strconv.ParseInt(strings.TrimSpace(sizeOut), 10, 64)
	}

	return FileInfo{
		Name:    baseName(objPath),
		IsDir:   false,
		Size:    size,
		ModTime: modTime,
	}, nil
}

// ReadDir lists the immediate children of the directory at the given path in the git ref.
func (g *GitFS) ReadDir(path string) ([]DirEntry, error) {
	objPath := path
	if objPath == "" || objPath == "." {
		objPath = ""
	}

	var lsPath string
	if objPath == "" {
		lsPath = ""
	} else {
		lsPath = objPath + "/"
	}

	// git ls-tree <ref> [<path>/] -- lists immediate children
	var out string
	var err error
	if lsPath == "" {
		out, err = g.git("ls-tree", g.ref)
	} else {
		out, err = g.git("ls-tree", g.ref, lsPath)
	}
	if err != nil {
		return nil, os.ErrNotExist
	}

	out = strings.TrimSpace(out)
	if out == "" {
		// Could be an empty tree or non-existent path
		return []DirEntry{}, nil
	}

	var entries []DirEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<mode> <type> <hash>\t<name>"
		tabIdx := strings.IndexByte(line, '\t')
		if tabIdx < 0 {
			continue
		}
		meta := line[:tabIdx]
		name := line[tabIdx+1:]

		fields := strings.Fields(meta)
		if len(fields) < 3 {
			continue
		}
		objType := fields[1]

		// Strip the path prefix to get the base name
		name = baseName(name)

		entries = append(entries, DirEntry{
			Name:  name,
			IsDir: objType == "tree",
		})
	}

	return entries, nil
}

func (g *GitFS) getModTime(path string) time.Time {
	var args []string
	if path == "." || path == "" {
		args = []string{"log", "-1", "--format=%ct", g.ref}
	} else {
		args = []string{"log", "-1", "--format=%ct", g.ref, "--", path}
	}
	out, err := g.git(args...)
	if err != nil {
		return time.Time{}
	}
	ts := strings.TrimSpace(out)
	if ts == "" {
		return time.Time{}
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}
