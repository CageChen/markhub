# MarkHub

[![CI](https://github.com/CageChen/markhub/actions/workflows/ci.yml/badge.svg)](https://github.com/CageChen/markhub/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/CageChen/markhub/branch/master/graph/badge.svg)](https://codecov.io/gh/CageChen/markhub)
![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.24-00ADD8.svg)

In the AI era, documentation lives across multiple repositories and branches — design docs on `feature-x`, API specs on `main`, architecture notes on `v2-refactor`. **MarkHub** brings them all into a single, beautiful web interface.

Point it at any number of local repos and git refs to instantly browse Markdown files with live hot-reload, syntax highlighting, and a premium reading experience. Zero configuration, single binary, just run and read.

## Features

- 📁 **Multi-Folder Support** - Serve multiple directories simultaneously with custom aliases
- 🌿 **Git Ref Browsing** - Browse any branch, tag, or commit without checking it out
- ✨ **GFM Support** - Full GitHub Flavored Markdown with tables, task lists, and more
- 🎨 **Syntax Highlighting** - Beautiful code blocks with automatic language detection
- 🔥 **Hot Reload** - Live updates via WebSocket when files change
- 🌓 **Dark / Light Theme** - Optimized light and dark themes
- 🧘 **Zen Mode** - Distraction-free reading (`Ctrl+Shift+Z`)
- 🚀 **Single Binary** - No external runtime required, all assets embedded

## Quick Start

```bash
git clone https://github.com/CageChen/markhub.git
cd markhub
make build
./bin/markhub --path ./docs --open
```

Or with Docker:

```bash
docker build -t markhub .
docker run -p 8080:8080 -v $(pwd)/docs:/docs markhub
```

## Configuration

MarkHub loads config from `~/.config/markhub/config.yaml` or `./markhub.yaml` (use `--config` to override):

```yaml
folders:
  - path: ./docs
    alias: Documentation
  - path: ./projects/web/notes
    alias: Web Notes
    exclude: ["drafts/**", "temp/**"]       # folder-level excludes
  - path: /home/user/my-repo
    alias: "my-repo (main)"
    git_ref: main                           # browse a git branch
    sub_path: docs                          # only serve a subdirectory
port: 8080
theme: dark
watch: true
extensions:
  - .md
  - .markdown

# global excludes — dependency dirs contain thousands of .md files from packages
exclude:
  - node_modules
  - .git
  - .svn
  - vendor

# repo-level excludes (applied to all refs of the same repo)
repo_exclude:
  /home/user/my-repo:
    - "internal/**"
```

Run `./bin/markhub --help` for all CLI options.

## License

MIT License - see [LICENSE](LICENSE) for details.
