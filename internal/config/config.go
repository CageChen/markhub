// Package config manages YAML-based configuration, CLI flags, and multi-folder settings.
package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Folder represents a folder with an alias for display
type Folder struct {
	Path    string   `yaml:"path" json:"path"`
	Alias   string   `yaml:"alias" json:"alias"`
	GitRef  string   `yaml:"git_ref,omitempty" json:"git_ref,omitempty"`
	SubPath string   `yaml:"sub_path,omitempty" json:"sub_path,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// Config holds all configuration options for MarkHub
type Config struct {
	// Legacy single path (for backward compatibility)
	Path string `yaml:"path,omitempty"`

	// Multiple folders with aliases
	Folders []Folder `yaml:"folders,omitempty" json:"folders"`

	Port       int      `yaml:"port"`
	Theme      string   `yaml:"theme"`
	Watch      bool     `yaml:"watch"`
	Open       bool     `yaml:"open"`
	Extensions []string `yaml:"extensions"`
	Exclude    []string `yaml:"exclude"`

	// Repo-level excludes keyed by absolute repo path
	RepoExclude map[string][]string `yaml:"repo_exclude,omitempty" json:"repo_exclude,omitempty"`

	// Internal: path to config file for saving
	configPath string
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Path:       ".",
		Port:       8080,
		Theme:      "light",
		Watch:      true,
		Open:       false,
		Extensions: []string{".md", ".markdown"},
		Exclude:    []string{"node_modules", ".git", ".svn"},
	}
}

// GetConfigDir returns the config directory path
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/markhub"
	}
	return filepath.Join(home, ".config", "markhub")
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

// Load loads configuration from file and command line flags
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Filter out 'serve' subcommand if present (for compatibility with `markhub serve --path`)
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "serve" {
		os.Args = append([]string{os.Args[0]}, args[1:]...)
	}

	// Define command line flags with sentinel values to detect if set
	path := flag.String("path", "", "Markdown files root directory")
	port := flag.Int("port", 0, "HTTP server port")
	theme := flag.String("theme", "", "Default theme (light/dark)")
	watch := flag.Bool("watch", true, "Enable file watching")
	open := flag.Bool("open", false, "Open browser on startup")
	configFile := flag.String("config", "", "Configuration file path")

	flag.StringVar(path, "p", "", "Markdown files root directory (shorthand)")

	flag.Parse()

	// Determine config file path
	var cfgPath string
	if *configFile != "" {
		cfgPath = *configFile
	} else {
		// Try ~/.config/markhub/config.yaml first
		globalConfig := GetConfigPath()
		if _, err := os.Stat(globalConfig); err == nil {
			cfgPath = globalConfig
		} else {
			// Fall back to local markhub.yaml
			if _, err := os.Stat("markhub.yaml"); err == nil {
				cfgPath = "markhub.yaml"
			}
		}
	}

	// Load from config file if found
	if cfgPath != "" {
		if err := cfg.loadFromFile(cfgPath); err != nil && *configFile != "" {
			// Only return error if user explicitly specified config file
			return nil, err
		}
		cfg.configPath = cfgPath
	} else {
		// Set default config path for saving
		cfg.configPath = GetConfigPath()
	}

	// Command line flags override config file (only if explicitly set)
	cliPathProvided := *path != ""
	if cliPathProvided {
		cfg.Path = *path
		// CLI --path overrides saved folders - use CLI path exclusively
		cfg.Folders = nil
	}
	if *port != 0 {
		cfg.Port = *port
	}
	if *theme != "" {
		cfg.Theme = *theme
	}
	// Bool flags - use command line value (they have explicit defaults)
	cfg.Watch = *watch
	cfg.Open = *open

	// Migrate legacy path to folders if needed
	cfg.migrateLegacyPath()

	return cfg, nil
}

// migrateLegacyPath converts single Path to Folders if Folders is empty
func (c *Config) migrateLegacyPath() {
	if len(c.Folders) == 0 && c.Path != "" {
		absPath, err := filepath.Abs(c.Path)
		if err != nil {
			absPath = c.Path
		}
		c.Folders = []Folder{{
			Path:  absPath,
			Alias: filepath.Base(absPath),
		}}
	}

	// Resolve all folder paths to absolute
	for i := range c.Folders {
		absPath, err := filepath.Abs(c.Folders[i].Path)
		if err == nil {
			c.Folders[i].Path = absPath
		}
		// Set alias to folder name if not specified
		if c.Folders[i].Alias == "" {
			c.Folders[i].Alias = filepath.Base(c.Folders[i].Path)
		}
	}
}

func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

// Save saves the current configuration to the config file
func (c *Config) Save() error {
	// Ensure config directory exists
	configDir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create a copy without internal fields for saving
	saveConfig := struct {
		Folders     []Folder            `yaml:"folders,omitempty"`
		Port        int                 `yaml:"port"`
		Theme       string              `yaml:"theme"`
		Watch       bool                `yaml:"watch"`
		Open        bool                `yaml:"open"`
		Extensions  []string            `yaml:"extensions"`
		Exclude     []string            `yaml:"exclude"`
		RepoExclude map[string][]string `yaml:"repo_exclude,omitempty"`
	}{
		Folders:     c.Folders,
		Port:        c.Port,
		Theme:       c.Theme,
		Watch:       c.Watch,
		Open:        c.Open,
		Extensions:  c.Extensions,
		Exclude:     c.Exclude,
		RepoExclude: c.RepoExclude,
	}

	data, err := yaml.Marshal(saveConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(c.configPath, data, 0644)
}

// AddFolder adds a new folder with the given path, alias, git_ref, subPath and excludes
func (c *Config) AddFolder(path, alias, gitRef, subPath string, exclude []string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Check if folder already exists (same path AND same git_ref AND same sub_path)
	for _, f := range c.Folders {
		if f.Path == absPath && f.GitRef == gitRef && f.SubPath == subPath {
			return nil // Already exists
		}
	}

	if alias == "" {
		alias = filepath.Base(absPath)
		if gitRef != "" {
			alias = alias + " (" + gitRef + ")"
		}
	}

	c.Folders = append(c.Folders, Folder{
		Path:    absPath,
		Alias:   alias,
		GitRef:  gitRef,
		SubPath: subPath,
		Exclude: exclude,
	})

	return nil
}

// IsFolderExcluded checks if a relative path should be excluded by folder-level excludes
func (c *Config) IsFolderExcluded(relPath string, folderExcludes []string) bool {
	if len(folderExcludes) == 0 {
		return false
	}
	for _, pattern := range folderExcludes {
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		base := filepath.Base(relPath)
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
		clean := filepath.Clean(pattern)
		if relPath == clean || strings.HasPrefix(relPath, clean+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// RemoveFolderByIndex removes a folder by its index
func (c *Config) RemoveFolderByIndex(index int) {
	if index < 0 || index >= len(c.Folders) {
		return
	}
	c.Folders = append(c.Folders[:index], c.Folders[index+1:]...)
}

// UpdateFolderByIndex updates a folder's fields by index
func (c *Config) UpdateFolderByIndex(index int, alias, gitRef, subPath string, exclude []string) {
	if index < 0 || index >= len(c.Folders) {
		return
	}
	c.Folders[index].Alias = alias
	c.Folders[index].GitRef = gitRef
	c.Folders[index].SubPath = subPath
	c.Folders[index].Exclude = exclude
}

// SetGlobalExclude sets the global exclude patterns
func (c *Config) SetGlobalExclude(patterns []string) {
	c.Exclude = patterns
}

// SetRepoExclude sets the exclude patterns for a specific repo path
func (c *Config) SetRepoExclude(repoPath string, patterns []string) {
	if c.RepoExclude == nil {
		c.RepoExclude = make(map[string][]string)
	}
	if len(patterns) == 0 {
		delete(c.RepoExclude, repoPath)
	} else {
		c.RepoExclude[repoPath] = patterns
	}
}

// GetRepoExclude returns the exclude patterns for a specific repo path
func (c *Config) GetRepoExclude(repoPath string) []string {
	if c.RepoExclude == nil {
		return nil
	}
	return c.RepoExclude[repoPath]
}

// GetConfigFilePath returns the path to the config file
func (c *Config) GetConfigFilePath() string {
	return c.configPath
}

// IsExcluded checks if a path should be excluded
func (c *Config) IsExcluded(path string) bool {
	base := filepath.Base(path)
	for _, exclude := range c.Exclude {
		if matched, _ := filepath.Match(exclude, base); matched {
			return true
		}
	}
	return false
}

// IsMarkdownFile checks if a file has a markdown extension
func (c *Config) IsMarkdownFile(path string) bool {
	ext := filepath.Ext(path)
	for _, e := range c.Extensions {
		if ext == e {
			return true
		}
	}
	return false
}
