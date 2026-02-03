package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Theme != "light" {
		t.Errorf("expected theme light, got %s", cfg.Theme)
	}
	if !cfg.Watch {
		t.Error("expected watch to be true")
	}
}

func TestMigrateLegacyPath(t *testing.T) {
	cfg := &Config{
		Path: "./test_docs",
	}
	cfg.migrateLegacyPath()

	if len(cfg.Folders) != 1 {
		t.Fatalf("expected 1 folder after migration, got %d", len(cfg.Folders))
	}

	absExpected, _ := filepath.Abs("./test_docs")
	if cfg.Folders[0].Path != absExpected {
		t.Errorf("expected path %s, got %s", absExpected, cfg.Folders[0].Path)
	}
	if cfg.Folders[0].Alias != "test_docs" {
		t.Errorf("expected alias test_docs, got %s", cfg.Folders[0].Alias)
	}
}

func TestAddFolder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Folders = nil

	err := cfg.AddFolder("./docs", "MyDocs", "", "", nil)
	if err != nil {
		t.Fatalf("AddFolder failed: %v", err)
	}

	if len(cfg.Folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(cfg.Folders))
	}

	if cfg.Folders[0].Alias != "MyDocs" {
		t.Errorf("expected alias MyDocs, got %s", cfg.Folders[0].Alias)
	}
}

func TestIsExcluded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Exclude = []string{".git", "node_modules"}

	if !cfg.IsExcluded("/path/to/.git") {
		t.Error("expected .git to be excluded")
	}
	if !cfg.IsExcluded("/path/to/node_modules") {
		t.Error("expected node_modules to be excluded")
	}
	if cfg.IsExcluded("/path/to/README.md") {
		t.Error("expected README.md NOT to be excluded")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	cfg := DefaultConfig()
	cfg.configPath = tmpFile
	cfg.Port = 9999
	cfg.Folders = []Folder{{Path: "/tmp", Alias: "Temp"}}

	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Manual load to verify
	cfg2 := &Config{}
	err = cfg2.loadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("loadFromFile failed: %v", err)
	}

	if cfg2.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg2.Port)
	}
	if len(cfg2.Folders) != 1 || cfg2.Folders[0].Alias != "Temp" {
		t.Errorf("folder loading failed")
	}
}
