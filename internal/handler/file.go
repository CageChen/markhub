// Package handler provides HTTP handlers for the MarkHub REST API.
package handler

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CageChen/markhub/internal/config"
	mfs "github.com/CageChen/markhub/internal/fs"
	"github.com/CageChen/markhub/internal/markdown"
	"github.com/gin-gonic/gin"
)

// FileResponse represents the response for a file request
type FileResponse struct {
	Path     string             `json:"path"`
	Title    string             `json:"title"`
	HTML     string             `json:"html"`
	TOC      []markdown.TOCItem `json:"toc"`
	ModTime  time.Time          `json:"modTime"`
	FolderID int                `json:"folderId"`
}

// FileHandler handles file content API requests
type FileHandler struct {
	cfg    *config.Config
	parser *markdown.Parser
}

// NewFileHandler creates a new file handler
func NewFileHandler(cfg *config.Config) *FileHandler {
	return &FileHandler{
		cfg:    cfg,
		parser: markdown.NewParser(),
	}
}

// resolvePath resolves a file path to its folder ID and relative path.
// Path format: {alias}/{relativePath} e.g., "markhub/docs/README.md"
func (h *FileHandler) resolvePath(filePath string) (mfs.FileSystem, string, int, error) {
	filePath = strings.TrimPrefix(filePath, "/")

	if filePath == "" {
		return nil, "", 0, os.ErrNotExist
	}

	var folderID int
	var relativePath string
	found := false

	parts := strings.SplitN(filePath, "/", 2)
	prefix := parts[0]
	if len(parts) > 1 {
		relativePath = parts[1]
	}

	// Match by folder alias
	for i, f := range h.cfg.Folders {
		if f.Alias == prefix {
			folderID = i
			found = true
			break
		}
	}

	if !found {
		return nil, "", 0, os.ErrNotExist
	}

	folder := h.cfg.Folders[folderID]

	// Security: prevent path traversal
	if strings.Contains(relativePath, "..") {
		return nil, "", 0, os.ErrPermission
	}

	fs := fsForFolder(folder)
	return fs, relativePath, folderID, nil
}

// GetFile returns the rendered HTML for a markdown file
func (h *FileHandler) GetFile(c *gin.Context) {
	filePath := c.Param("path")
	if filePath == "" {
		filePath = c.Query("path")
	}

	// Security: prevent path traversal
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "invalid path",
		})
		return
	}

	fs, relativePath, folderID, err := h.resolvePath(filePath)
	if err != nil {
		status := http.StatusBadRequest
		msg := err.Error()
		if os.IsNotExist(err) {
			status = http.StatusNotFound
			msg = "file not found"
		} else if os.IsPermission(err) {
			status = http.StatusForbidden
			msg = "access denied"
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}

	// Check if file exists and is not a directory
	info, err := fs.Stat(relativePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "file not found",
		})
		return
	}

	if info.IsDir {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "path is a directory",
		})
		return
	}

	// Read and parse the file
	content, err := fs.ReadFile(relativePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "file not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to read file: %v", err),
		})
		return
	}

	result, err := h.parser.Parse(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse markdown: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, FileResponse{
		Path:     strings.TrimPrefix(filePath, "/"),
		Title:    result.Title,
		HTML:     result.HTML,
		TOC:      result.TOC,
		ModTime:  info.ModTime,
		FolderID: folderID,
	})
}

// GetRaw returns the raw markdown content
func (h *FileHandler) GetRaw(c *gin.Context) {
	filePath := c.Param("path")

	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "invalid path",
		})
		return
	}

	fs, relativePath, _, err := h.resolvePath(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "file not found",
			})
		} else {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "access denied",
			})
		}
		return
	}

	content, err := fs.ReadFile(relativePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "file not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to read file: %v", err),
		})
		return
	}

	c.Data(http.StatusOK, "text/markdown; charset=utf-8", content)
}
