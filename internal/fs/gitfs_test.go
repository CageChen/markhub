package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository with sample files for testing.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	git("init")
	git("config", "user.email", "test@test.com")
	git("config", "user.name", "Test")

	// Create files and directories
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Guide\n\nHello world.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	git("add", "-A")
	git("commit", "-m", "initial commit")

	return dir
}

func TestGitFS_Stat_Root(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewGitFS(dir, "HEAD")

	info, err := g.Stat("")
	if err != nil {
		t.Fatalf("Stat('') failed: %v", err)
	}
	if !info.IsDir {
		t.Error("expected root to be a directory")
	}
}

func TestGitFS_ReadDir_Root(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewGitFS(dir, "HEAD")

	entries, err := g.ReadDir("")
	if err != nil {
		t.Fatalf("ReadDir('') failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected non-empty root directory")
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
		t.Logf("  entry: %s (dir=%v)", e.Name, e.IsDir)
	}

	if !names["README.md"] {
		t.Error("expected README.md in root entries")
	}
	if !names["docs"] {
		t.Error("expected docs directory in root entries")
	}
}

func TestGitFS_Stat_Dir(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewGitFS(dir, "HEAD")

	info, err := g.Stat("docs")
	if err != nil {
		t.Fatalf("Stat('docs') failed: %v", err)
	}
	if !info.IsDir {
		t.Error("expected docs to be a directory")
	}
	if info.Name != "docs" {
		t.Errorf("expected name 'docs', got %q", info.Name)
	}
}

func TestGitFS_ReadDir_SubDir(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewGitFS(dir, "HEAD")

	entries, err := g.ReadDir("docs")
	if err != nil {
		t.Fatalf("ReadDir('docs') failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry in docs, got %d", len(entries))
	}
	if entries[0].Name != "guide.md" {
		t.Errorf("expected guide.md, got %s", entries[0].Name)
	}
}

func TestGitFS_ReadFile(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewGitFS(dir, "HEAD")

	content, err := g.ReadFile("docs/guide.md")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected non-empty file content")
	}
	t.Logf("content: %s", content)
}

func TestGitFS_ReadFile_NotExist(t *testing.T) {
	dir := setupTestRepo(t)
	g := NewGitFS(dir, "HEAD")

	_, err := g.ReadFile("nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
