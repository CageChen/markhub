// Package main is the entry point for the MarkHub server.
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/CageChen/markhub/internal/config"
	"github.com/CageChen/markhub/internal/handler"
	"github.com/CageChen/markhub/internal/watcher"
	"github.com/gin-gonic/gin"
)

//go:embed web/*
var webFS embed.FS

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("MarkHub - Markdown Renderer")
	log.Printf("Config file: %s", cfg.GetConfigFilePath())
	log.Printf("Serving %d folder(s):", len(cfg.Folders))
	for i, f := range cfg.Folders {
		if f.GitRef != "" {
			log.Printf("  [%d] %s -> %s (git ref: %s)", i, f.Alias, f.Path, f.GitRef)
		} else {
			log.Printf("  [%d] %s -> %s", i, f.Alias, f.Path)
		}
	}
	log.Printf("Server starting at: http://localhost:%d", cfg.Port)

	// Create handlers
	treeHandler := handler.NewTreeHandler(cfg)
	fileHandler := handler.NewFileHandler(cfg)
	wsHandler := handler.NewWSHandler()

	// Setup file watcher if enabled
	if cfg.Watch {
		w, err := watcher.New(cfg)
		if err != nil {
			log.Printf("Warning: failed to create file watcher: %v", err)
		} else {
			w.OnChange(wsHandler.OnFileChange)
			if err := w.Start(); err != nil {
				log.Printf("Warning: failed to start file watcher: %v", err)
			}
			defer func() { _ = w.Stop() }()
			log.Printf("File watcher enabled")
		}
	}

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// API routes
	api := r.Group("/api")
	{
		// Tree and file APIs
		api.GET("/tree", treeHandler.GetTree)
		api.GET("/files/*path", fileHandler.GetFile)
		api.GET("/raw/*path", fileHandler.GetRaw)
		api.GET("/ws", wsHandler.HandleWS)

		// Folder management APIs
		api.GET("/folders", treeHandler.GetFolders)
		api.POST("/folders", treeHandler.AddFolder)
		api.PUT("/folders", treeHandler.UpdateFolder)
		api.DELETE("/folders", treeHandler.RemoveFolder)
		api.PUT("/exclude", treeHandler.UpdateGlobalExclude)
		api.PUT("/repo-exclude", treeHandler.UpdateRepoExclude)
	}

	// Serve embedded static files
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to load web assets: %v", err)
	}
	r.NoRoute(gin.WrapH(http.FileServer(http.FS(webContent))))

	// Open browser if requested
	if cfg.Open {
		go openBrowser(fmt.Sprintf("http://localhost:%d", cfg.Port))
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	_ = exec.Command(cmd, args...).Start()
}
