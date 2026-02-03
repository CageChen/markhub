package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/CageChen/markhub/internal/config"
	mfs "github.com/CageChen/markhub/internal/fs"
	"github.com/gin-gonic/gin"
)

// TreeNode represents a file or directory in the tree
type TreeNode struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Path        string      `json:"path,omitempty"`
	Alias       string      `json:"alias,omitempty"`
	FolderID    int         `json:"folderId,omitempty"`
	Children    []*TreeNode `json:"children,omitempty"`
	ModTime     *time.Time  `json:"modTime,omitempty"`
	Size        int64       `json:"size,omitempty"`
	IsRepoGroup bool        `json:"isRepoGroup,omitempty"`
}

// TreeHandler handles directory tree API requests
type TreeHandler struct {
	cfg *config.Config
}

// NewTreeHandler creates a new tree handler
func NewTreeHandler(cfg *config.Config) *TreeHandler {
	return &TreeHandler{cfg: cfg}
}

// fsForFolder returns the appropriate FileSystem for a folder config.
func fsForFolder(folder config.Folder) mfs.FileSystem {
	if folder.GitRef != "" {
		return mfs.NewGitFS(folder.Path, folder.GitRef)
	}
	return mfs.NewLocalFS(folder.Path)
}

// GetTree returns the directory tree structure for all configured folders
func (h *TreeHandler) GetTree(c *gin.Context) {
	var rawRoots []*TreeNode

	for i, folder := range h.cfg.Folders {
		fs := fsForFolder(folder)
		// Merge repo-level excludes with folder-level excludes
		mergedExcludes := append([]string{}, h.cfg.GetRepoExclude(folder.Path)...)
		mergedExcludes = append(mergedExcludes, folder.Exclude...)
		tree, err := h.buildTree(fs, folder.SubPath, i, folder.Alias, mergedExcludes)
		if err != nil {
			continue
		}
		tree.Name = folder.Alias
		tree.Alias = folder.Alias
		tree.FolderID = i
		rawRoots = append(rawRoots, tree)
	}

	// Group folders that share the same path and have git_ref set
	roots := h.groupByRepo(rawRoots)

	if len(roots) == 1 {
		c.JSON(http.StatusOK, roots[0])
	} else {
		c.JSON(http.StatusOK, gin.H{
			"type":     "root",
			"children": roots,
		})
	}
}

// groupByRepo groups folder roots that share the same filesystem path (i.e.
// multiple git refs of the same repo) under a single parent node named after
// the repository directory.  Folders without a GitRef are kept as-is.
func (h *TreeHandler) groupByRepo(roots []*TreeNode) []*TreeNode {
	// Build a map: repoPath -> []folderIndex for folders that have GitRef
	type entry struct {
		folderIdx int
		node      *TreeNode
	}
	repoMap := make(map[string][]entry)
	var order []string // preserve first-seen order of repo paths
	var standalone []*TreeNode

	for _, node := range roots {
		if node.FolderID >= len(h.cfg.Folders) {
			standalone = append(standalone, node)
			continue
		}
		folder := h.cfg.Folders[node.FolderID]
		if folder.GitRef == "" {
			standalone = append(standalone, node)
			continue
		}
		if _, seen := repoMap[folder.Path]; !seen {
			order = append(order, folder.Path)
		}
		repoMap[folder.Path] = append(repoMap[folder.Path], entry{folderIdx: node.FolderID, node: node})
	}

	var result []*TreeNode

	// Emit grouped repos in order
	for _, repoPath := range order {
		entries := repoMap[repoPath]
		if len(entries) == 1 {
			// Single ref for this repo — no grouping needed
			result = append(result, entries[0].node)
			continue
		}
		// Create a virtual parent node for the repo
		groupNode := &TreeNode{
			Name:        filepath.Base(repoPath),
			Type:        "directory",
			IsRepoGroup: true,
		}
		for _, e := range entries {
			groupNode.Children = append(groupNode.Children, e.node)
		}
		result = append(result, groupNode)
	}

	result = append(result, standalone...)
	return result
}

// folderResponse extends a Folder with computed effective excludes for the frontend.
type folderResponse struct {
	config.Folder
	EffectiveExcludes []string `json:"effective_excludes"`
}

// GetFolders returns the list of configured folders, global excludes, and repo excludes
func (h *TreeHandler) GetFolders(c *gin.Context) {
	resp := make([]folderResponse, len(h.cfg.Folders))
	for i, f := range h.cfg.Folders {
		merged := append([]string{}, h.cfg.GetRepoExclude(f.Path)...)
		merged = append(merged, f.Exclude...)
		resp[i] = folderResponse{Folder: f, EffectiveExcludes: merged}
	}
	c.JSON(http.StatusOK, gin.H{
		"folders":       resp,
		"globalExclude": h.cfg.Exclude,
		"repoExclude":   h.cfg.RepoExclude,
	})
}

// AddFolderRequest represents a request to add a folder
type AddFolderRequest struct {
	Path    string   `json:"path" binding:"required"`
	Alias   string   `json:"alias"`
	GitRef  string   `json:"git_ref"`
	SubPath string   `json:"sub_path"`
	Exclude []string `json:"exclude"`
}

// AddFolder adds a new folder to the configuration
func (h *TreeHandler) AddFolder(c *gin.Context) {
	var req AddFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "path is required",
		})
		return
	}

	// Validate path exists (it must be a directory on disk even for git_ref folders)
	info, err := os.Stat(req.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "path does not exist: " + req.Path,
		})
		return
	}
	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "path is not a directory",
		})
		return
	}

	// Validate SubPath if provided
	if req.SubPath != "" {
		fs := fsForFolder(config.Folder{Path: req.Path, GitRef: req.GitRef})
		if _, err := fs.Stat(req.SubPath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "sub_path does not exist: " + req.SubPath,
			})
			return
		}
	}

	// Add folder
	if err := h.cfg.AddFolder(req.Path, req.Alias, req.GitRef, req.SubPath, req.Exclude); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Save configuration
	if err := h.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "folder added",
		"folders": h.cfg.Folders,
	})
}

// UpdateFolderRequest represents a request to update a folder (identified by index)
type UpdateFolderRequest struct {
	Index   int      `json:"index"`
	Alias   string   `json:"alias" binding:"required"`
	GitRef  string   `json:"git_ref"`
	SubPath string   `json:"sub_path"`
	Exclude []string `json:"exclude"`
}

// UpdateFolder updates a folder's settings by index
func (h *TreeHandler) UpdateFolder(c *gin.Context) {
	var req UpdateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "alias is required",
		})
		return
	}

	if req.Index < 0 || req.Index >= len(h.cfg.Folders) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid folder index",
		})
		return
	}

	h.cfg.UpdateFolderByIndex(req.Index, req.Alias, req.GitRef, req.SubPath, req.Exclude)

	// Save configuration
	if err := h.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "folder updated",
		"folders": h.cfg.Folders,
	})
}

// RemoveFolderRequest represents a request to remove a folder (by index)
type RemoveFolderRequest struct {
	Index int `json:"index"`
}

// RemoveFolder removes a folder from the configuration by index
func (h *TreeHandler) RemoveFolder(c *gin.Context) {
	var req RemoveFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "index is required",
		})
		return
	}

	if req.Index < 0 || req.Index >= len(h.cfg.Folders) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid folder index",
		})
		return
	}

	h.cfg.RemoveFolderByIndex(req.Index)

	// Save configuration
	if err := h.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "folder removed",
		"folders": h.cfg.Folders,
	})
}

// UpdateRepoExcludeRequest represents a request to update repo-level excludes
type UpdateRepoExcludeRequest struct {
	Path    string   `json:"path" binding:"required"`
	Exclude []string `json:"exclude"`
}

// UpdateRepoExclude updates the exclude patterns for a specific repo path
func (h *TreeHandler) UpdateRepoExclude(c *gin.Context) {
	var req UpdateRepoExcludeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "path is required",
		})
		return
	}

	h.cfg.SetRepoExclude(req.Path, req.Exclude)

	if err := h.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "repo excludes updated",
		"repoExclude": h.cfg.RepoExclude,
	})
}

// UpdateGlobalExcludeRequest represents a request to update global excludes
type UpdateGlobalExcludeRequest struct {
	Exclude []string `json:"exclude"`
}

// UpdateGlobalExclude updates the global exclude patterns
func (h *TreeHandler) UpdateGlobalExclude(c *gin.Context) {
	var req UpdateGlobalExcludeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	h.cfg.SetGlobalExclude(req.Exclude)

	if err := h.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "global excludes updated",
		"globalExclude": h.cfg.Exclude,
	})
}

func (h *TreeHandler) buildTree(
	fs mfs.FileSystem, relativePath string, folderID int, folderAlias string, folderExcludes []string,
) (*TreeNode, error) {
	info, err := fs.Stat(relativePath)
	if err != nil {
		return nil, err
	}

	// Build path with folder alias prefix for stable, human-readable URLs
	nodePath := relativePath
	if relativePath != "" {
		nodePath = folderAlias + "/" + relativePath
	}

	node := &TreeNode{
		Name:     info.Name,
		Path:     nodePath,
		FolderID: folderID,
	}

	if info.IsDir {
		node.Type = "directory"
		entries, err := fs.ReadDir(relativePath)
		if err != nil {
			return nil, err
		}

		// Sort: directories first, then files, both alphabetically
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir != entries[j].IsDir {
				return entries[i].IsDir
			}
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})

		for _, entry := range entries {
			name := entry.Name
			childPath := relativePath
			if childPath == "" {
				childPath = name
			} else {
				childPath = childPath + "/" + name
			}

			// Skip globally excluded paths
			if h.cfg.IsExcluded(name) {
				continue
			}

			// Skip folder-level excluded paths
			if h.cfg.IsFolderExcluded(childPath, folderExcludes) {
				continue
			}

			// Skip non-markdown files (but include directories)
			if !entry.IsDir && !h.cfg.IsMarkdownFile(name) {
				continue
			}

			child, err := h.buildTree(fs, childPath, folderID, folderAlias, folderExcludes)
			if err != nil {
				continue
			}

			// Skip empty directories
			if child.Type == "directory" && len(child.Children) == 0 {
				continue
			}

			node.Children = append(node.Children, child)
		}
	} else {
		node.Type = "file"
		modTime := info.ModTime
		node.ModTime = &modTime
		node.Size = info.Size
	}

	return node, nil
}

// ToJSON converts the tree to JSON for SSE
func (n *TreeNode) ToJSON() string {
	data, _ := json.Marshal(n)
	return string(data)
}
