/**
 * MarkHub - Frontend Application
 * Modern Markdown renderer with hot reload and multi-folder support
 */

class MarkHub {
    constructor() {
        this.currentPath = null;
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.folders = [];
        this.editingFolderIndex = null;
        this.editingRepoExclude = null;
        this.zenMode = false;

        this.init();
    }

    async init() {
        this.initTheme();
        this.initZenMode();
        this.bindEvents();
        await this.loadFileTree();
        this.initWebSocket();
        this.handleInitialRoute();
    }

    // ========================================
    // Theme Management
    // ========================================
    initTheme() {
        const savedTheme = localStorage.getItem('markhub-theme') || 'light';
        document.documentElement.dataset.theme = savedTheme;
    }

    toggleTheme() {
        const current = document.documentElement.dataset.theme;
        const next = current === 'dark' ? 'light' : 'dark';
        document.documentElement.dataset.theme = next;
        localStorage.setItem('markhub-theme', next);
    }

    // ========================================
    // Zen Mode
    // ========================================
    initZenMode() {
        const saved = localStorage.getItem('markhub-zen');
        if (saved === 'true') {
            this.zenMode = true;
            document.querySelector('.app').classList.add('zen-mode');
        }
    }

    toggleZenMode() {
        this.zenMode = !this.zenMode;
        document.querySelector('.app').classList.toggle('zen-mode', this.zenMode);
        localStorage.setItem('markhub-zen', this.zenMode);
    }

    // ========================================
    // Event Bindings
    // ========================================
    bindEvents() {
        // Theme toggle
        document.getElementById('themeToggle').addEventListener('click', () => {
            this.toggleTheme();
        });

        // Settings/folder management
        document.getElementById('settingsBtn').addEventListener('click', () => {
            this.showFolderModal();
        });

        document.getElementById('modalClose').addEventListener('click', () => {
            this.hideFolderModal();
        });

        document.getElementById('folderModal').addEventListener('click', (e) => {
            if (e.target.id === 'folderModal') {
                this.hideFolderModal();
            }
        });

        document.getElementById('addFolderBtn').addEventListener('click', () => {
            this.addFolder();
        });

        document.getElementById('saveGlobalExcludeBtn').addEventListener('click', () => {
            this.saveGlobalExclude();
        });

        // Sidebar toggle (mobile)
        document.getElementById('sidebarToggle').addEventListener('click', () => {
            document.getElementById('sidebar').classList.toggle('open');
        });

        // Search input
        const searchInput = document.getElementById('searchInput');
        searchInput.addEventListener('input', (e) => {
            this.filterTree(e.target.value);
        });

        // Handle browser navigation
        window.addEventListener('popstate', (e) => {
            if (e.state && e.state.path) {
                this.loadFile(e.state.path, false);
            }
        });

        // Close sidebar on mobile when clicking content
        document.querySelector('.main-content').addEventListener('click', () => {
            if (window.innerWidth <= 768) {
                document.getElementById('sidebar').classList.remove('open');
            }
        });

        // TOC scroll spy
        window.addEventListener('scroll', () => {
            this.updateActiveTocItem();
        });

        // Zen mode toggle
        document.getElementById('zenToggle').addEventListener('click', () => {
            this.toggleZenMode();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Ctrl+Shift+Z (or Cmd+Shift+Z on Mac) toggles zen mode
            if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'Z') {
                e.preventDefault();
                this.toggleZenMode();
                return;
            }
            if (e.key === 'Escape') {
                if (this.zenMode) {
                    this.toggleZenMode();
                } else {
                    this.hideFolderModal();
                }
            }
        });
    }

    // ========================================
    // Folder Management Modal
    // ========================================
    async showFolderModal() {
        await this.loadFolders();
        this.renderFolderList();
        this.renderGlobalExclude();
        document.getElementById('folderModal').classList.add('visible');
    }

    hideFolderModal() {
        this.editingFolderIndex = null;
        this.editingRepoExclude = null;
        document.getElementById('folderModal').classList.remove('visible');
        document.getElementById('folderPath').value = '';
        document.getElementById('folderAlias').value = '';
        document.getElementById('folderGitRef').value = '';
        document.getElementById('folderSubPath').value = '';
        document.getElementById('folderExclude').value = '';
    }

    async loadFolders() {
        try {
            const response = await fetch('/api/folders');
            if (response.ok) {
                const data = await response.json();
                this.folders = data.folders || [];
                this.globalExclude = data.globalExclude || [];
                this.repoExclude = data.repoExclude || {};
            }
        } catch (error) {
            console.error('Failed to load folders:', error);
        }
    }

    renderGlobalExclude() {
        const textarea = document.getElementById('globalExclude');
        textarea.value = (this.globalExclude || []).join('\n');
    }

    renderFolderList() {
        const container = document.getElementById('folderList');

        if (this.folders.length === 0) {
            container.innerHTML = '<div class="empty-folders">No folders configured</div>';
            return;
        }

        // Group folders by path for those that have git_ref
        const repoGroups = {};
        const repoOrder = [];
        const standalone = [];

        this.folders.forEach((folder, index) => {
            if (folder.git_ref) {
                if (!repoGroups[folder.path]) {
                    repoGroups[folder.path] = [];
                    repoOrder.push(folder.path);
                }
                repoGroups[folder.path].push({ folder, index });
            } else {
                standalone.push({ folder, index });
            }
        });

        let html = '';

        // Render grouped repos
        for (const repoPath of repoOrder) {
            const entries = repoGroups[repoPath];
            const repoName = repoPath.split('/').pop();
            const repoExcludes = (this.repoExclude || {})[repoPath] || [];
            const repoExcludeCount = repoExcludes.length;
            const isEditingRepoExclude = this.editingRepoExclude === repoPath;

            html += `<div class="repo-group-card">`;
            html += `<div class="repo-group-header">`;
            html += `<div class="repo-group-title">`;
            html += `<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>`;
            html += `<span>${this.escapeHtml(repoName)}</span>`;
            html += `</div>`;
            html += `<div class="repo-group-path">${this.escapeHtml(repoPath)}</div>`;

            if (isEditingRepoExclude) {
                const escapedPath = this.escapeHtml(repoPath).replace(/'/g, "\\'");
                html += `<div class="folder-edit-form" data-repo-exclude-path="${this.escapeHtml(repoPath)}">`;
                html += `<div class="form-group"><label>Repo Excludes <span class="label-hint">(comma-separated, applied to all refs)</span></label>`;
                html += `<input type="text" id="editRepoExcludeInput" value="${this.escapeHtml(repoExcludes.join(', '))}" placeholder="e.g. vendor/*, docs/*"></div>`;
                html += `<div class="folder-edit-actions">`;
                html += `<button class="btn btn-primary btn-sm" onclick="markhub.saveRepoExclude('${escapedPath}')">Save</button>`;
                html += `<button class="btn btn-secondary btn-sm" onclick="markhub.cancelRepoExclude()">Cancel</button>`;
                html += `</div></div>`;
            } else {
                const repoExcludeInfo = repoExcludes.length > 0
                    ? `<span class="repo-exclude-values">${repoExcludes.map(e => `<span class="exclude-tag">${this.escapeHtml(e)}</span>`).join(' ')}</span>`
                    : `<span class="repo-exclude-values empty">None</span>`;

                html += `<details class="repo-exclude-details">`;
                html += `<summary class="repo-exclude-summary">Shared Repo Excludes (${repoExcludeCount})`;
                html += `<button class="btn-edit-inline" title="Edit repo excludes" onclick="event.preventDefault(); markhub.editRepoExclude('${this.escapeHtml(repoPath).replace(/'/g, "\\'")}')">`;
                html += `<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor"><path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/></svg>`;
                html += `</button>`;
                html += `</summary>`;
                html += `<div class="repo-exclude-body">${repoExcludeInfo}</div>`;
                html += `</details>`;
            }

            html += `</div>`;

            for (const { folder, index } of entries) {
                if (this.editingFolderIndex === index) {
                    html += this.renderFolderEditForm(folder, index, true);
                } else {
                    const subPathBadge = folder.sub_path
                        ? ` <span class="badge badge-subpath" title="Sub path">${this.escapeHtml(folder.sub_path)}</span>`
                        : '';
                    const excludeTags = folder.exclude && folder.exclude.length > 0
                        ? `<div class="folder-exclude-tags">${folder.exclude.map(e => `<span class="exclude-tag">${this.escapeHtml(e)}</span>`).join(' ')}</div>`
                        : `<div class="folder-exclude-tags"><span class="effective-excludes-hint">No ref-specific excludes</span></div>`;
                    html += `
                    <div class="folder-item folder-item-grouped" data-index="${index}">
                        <svg viewBox="0 0 24 24" width="20" height="20" fill="var(--warning)">
                            <path d="M10 4H4a2 2 0 00-2 2v12a2 2 0 002 2h16a2 2 0 002-2V8a2 2 0 00-2-2h-8l-2-2z"/>
                        </svg>
                        <div class="folder-info">
                            <div class="folder-alias">${this.escapeHtml(folder.alias)} <span class="badge badge-git" title="Git ref">${this.escapeHtml(folder.git_ref)}</span>${subPathBadge}</div>
                            ${excludeTags}
                        </div>
                        <div class="folder-actions">
                            <button class="btn-edit" title="Edit" onclick="markhub.editFolder(${index})">
                                <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                                    <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
                                </svg>
                            </button>
                            <button class="btn-delete" title="Remove folder" onclick="markhub.removeFolder(${index})">
                                <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                                    <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                                </svg>
                            </button>
                        </div>
                    </div>`;
                }
            }

            html += `</div>`;
        }

        // Render standalone folders
        for (const { folder, index } of standalone) {
            if (this.editingFolderIndex === index) {
                html += this.renderFolderEditForm(folder, index, false);
            } else {
                const subPathBadge = folder.sub_path
                    ? ` <span class="badge badge-subpath" title="Sub path">${this.escapeHtml(folder.sub_path)}</span>`
                    : '';
                const excludeInfo = folder.exclude && folder.exclude.length > 0
                    ? `<div class="folder-exclude">Excludes: ${folder.exclude.map(e => this.escapeHtml(e)).join(', ')}</div>`
                    : '';
                html += `
                <div class="folder-item" data-index="${index}">
                    <svg viewBox="0 0 24 24" width="24" height="24" fill="var(--warning)">
                        <path d="M10 4H4a2 2 0 00-2 2v12a2 2 0 002 2h16a2 2 0 002-2V8a2 2 0 00-2-2h-8l-2-2z"/>
                    </svg>
                    <div class="folder-info">
                        <div class="folder-alias">${this.escapeHtml(folder.alias)}${subPathBadge}</div>
                        <div class="folder-path">${this.escapeHtml(folder.path)}</div>
                        ${excludeInfo}
                    </div>
                    <div class="folder-actions">
                        <button class="btn-edit" title="Edit" onclick="markhub.editFolder(${index})">
                            <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                                <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
                            </svg>
                        </button>
                        <button class="btn-delete" title="Remove folder" onclick="markhub.removeFolder(${index})">
                            <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                                <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                            </svg>
                        </button>
                    </div>
                </div>`;
            }
        }

        container.innerHTML = html;
    }

    renderFolderEditForm(folder, index, isGrouped) {
        const cssClass = isGrouped ? 'folder-edit-form folder-item-grouped' : 'folder-edit-form';
        let html = `<div class="${cssClass}" data-index="${index}">`;
        html += `<div class="form-group"><label>Display Name</label>`;
        html += `<input type="text" id="editFolderAlias" value="${this.escapeHtml(folder.alias)}" placeholder="Display name"></div>`;
        if (folder.git_ref !== undefined) {
            html += `<div class="form-group"><label>Git Ref <span class="label-hint">(branch, tag, or commit)</span></label>`;
            html += `<input type="text" id="editFolderGitRef" value="${this.escapeHtml(folder.git_ref || '')}" placeholder="e.g. main, v1.0"></div>`;
            html += `<div class="form-group"><label>Sub Path <span class="label-hint">(subdirectory within repo)</span></label>`;
            html += `<input type="text" id="editFolderSubPath" value="${this.escapeHtml(folder.sub_path || '')}" placeholder="e.g. docs/"></div>`;
        }
        html += `<div class="form-group"><label>Excludes <span class="label-hint">(comma-separated)</span></label>`;
        html += `<input type="text" id="editFolderExclude" value="${this.escapeHtml((folder.exclude || []).join(', '))}" placeholder="e.g. vendor/*, node_modules/*"></div>`;
        html += `<div class="folder-edit-actions">`;
        html += `<button class="btn btn-primary btn-sm" onclick="markhub.saveEditFolder(${index})">Save</button>`;
        html += `<button class="btn btn-secondary btn-sm" onclick="markhub.cancelEditFolder()">Cancel</button>`;
        html += `</div></div>`;
        return html;
    }

    async addFolder() {
        const pathInput = document.getElementById('folderPath');
        const aliasInput = document.getElementById('folderAlias');
        const gitRefInput = document.getElementById('folderGitRef');
        const subPathInput = document.getElementById('folderSubPath');
        const excludeInput = document.getElementById('folderExclude');
        const path = pathInput.value.trim();
        const alias = aliasInput.value.trim();
        const git_ref = gitRefInput.value.trim();
        const sub_path = subPathInput.value.trim();
        const excludeStr = excludeInput.value.trim();
        const exclude = excludeStr ? excludeStr.split(',').map(s => s.trim()).filter(Boolean) : [];

        if (!path) {
            alert('Please enter a folder path');
            return;
        }

        try {
            const response = await fetch('/api/folders', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path, alias, git_ref, sub_path, exclude })
            });

            const data = await response.json();

            if (response.ok) {
                this.folders = data.folders || [];
                this.renderFolderList();
                pathInput.value = '';
                aliasInput.value = '';
                gitRefInput.value = '';
                subPathInput.value = '';
                excludeInput.value = '';
                await this.loadFileTree();
            } else {
                alert(data.error || 'Failed to add folder');
            }
        } catch (error) {
            console.error('Failed to add folder:', error);
            alert('Failed to add folder');
        }
    }

    editFolder(index) {
        const folder = this.folders[index];
        if (!folder) return;

        this.editingFolderIndex = index;
        this.editingRepoExclude = null;
        this.renderFolderList();
    }

    async saveEditFolder(index) {
        const folder = this.folders[index];
        if (!folder) return;

        const aliasEl = document.getElementById('editFolderAlias');
        const gitRefEl = document.getElementById('editFolderGitRef');
        const subPathEl = document.getElementById('editFolderSubPath');
        const excludeEl = document.getElementById('editFolderExclude');

        const alias = aliasEl ? aliasEl.value.trim() : folder.alias;
        const git_ref = gitRefEl ? gitRefEl.value.trim() : (folder.git_ref || '');
        const sub_path = subPathEl ? subPathEl.value.trim() : (folder.sub_path || '');
        const excludeStr = excludeEl ? excludeEl.value.trim() : '';
        const exclude = excludeStr ? excludeStr.split(',').map(s => s.trim()).filter(Boolean) : [];

        try {
            const response = await fetch('/api/folders', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ index, alias, git_ref, sub_path, exclude })
            });

            const data = await response.json();

            if (response.ok) {
                this.folders = data.folders || [];
                this.editingFolderIndex = null;
                this.renderFolderList();
                await this.loadFileTree();
            } else {
                alert(data.error || 'Failed to update folder');
            }
        } catch (error) {
            console.error('Failed to update folder:', error);
        }
    }

    cancelEditFolder() {
        this.editingFolderIndex = null;
        this.renderFolderList();
    }

    async removeFolder(index) {
        if (!confirm('Remove this folder from MarkHub?')) return;

        try {
            const response = await fetch('/api/folders', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ index })
            });

            const data = await response.json();

            if (response.ok) {
                this.folders = data.folders || [];
                this.renderFolderList();
                await this.loadFileTree();
            } else {
                alert(data.error || 'Failed to remove folder');
            }
        } catch (error) {
            console.error('Failed to remove folder:', error);
        }
    }

    async saveGlobalExclude() {
        const textarea = document.getElementById('globalExclude');
        const patterns = textarea.value.split('\n').map(s => s.trim()).filter(Boolean);

        try {
            const response = await fetch('/api/exclude', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ exclude: patterns })
            });

            const data = await response.json();

            if (response.ok) {
                this.globalExclude = data.globalExclude || [];
                await this.loadFileTree();
            } else {
                alert(data.error || 'Failed to update global excludes');
            }
        } catch (error) {
            console.error('Failed to update global excludes:', error);
        }
    }

    editRepoExclude(repoPath) {
        this.editingRepoExclude = repoPath;
        this.editingFolderIndex = null;
        this.renderFolderList();
    }

    async saveRepoExclude(repoPath) {
        const input = document.getElementById('editRepoExcludeInput');
        const excludeStr = input ? input.value.trim() : '';
        const exclude = excludeStr ? excludeStr.split(',').map(s => s.trim()).filter(Boolean) : [];

        try {
            const response = await fetch('/api/repo-exclude', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path: repoPath, exclude })
            });

            const data = await response.json();

            if (response.ok) {
                this.repoExclude = data.repoExclude || {};
                this.editingRepoExclude = null;
                this.renderFolderList();
                await this.loadFileTree();
            } else {
                alert(data.error || 'Failed to update repo excludes');
            }
        } catch (error) {
            console.error('Failed to update repo excludes:', error);
        }
    }

    cancelRepoExclude() {
        this.editingRepoExclude = null;
        this.renderFolderList();
    }

    // ========================================
    // File Tree
    // ========================================
    async loadFileTree() {
        try {
            const response = await fetch('/api/tree');
            if (!response.ok) throw new Error('Failed to load file tree');
            const tree = await response.json();
            this.renderFileTree(tree);
        } catch (error) {
            console.error('Error loading file tree:', error);
            document.getElementById('fileTree').innerHTML = `
                <div class="loading" style="color: var(--error)">
                    Failed to load files
                </div>
            `;
        }
    }

    renderFileTree(node, container = null) {
        if (!container) {
            container = document.getElementById('fileTree');
            container.innerHTML = '';
        }

        // Handle root with multiple folders
        if (node.type === 'root' && node.children) {
            node.children.forEach(child => this.renderFileTree(child, container));
            return;
        }

        // Handle single folder or directory
        if (node.type === 'directory') {
            const item = this.createTreeItem(node, true, node.alias !== undefined);
            container.appendChild(item);

            if (node.children) {
                const childrenContainer = item.querySelector('.tree-children');
                node.children.forEach(child => this.renderFileTree(child, childrenContainer));
            }
        } else if (node.type === 'file') {
            container.appendChild(this.createTreeItem(node, false, false));
        }
    }

    createTreeItem(node, isDir, isRootFolder = false) {
        const item = document.createElement('div');
        const isRepoGroup = node.isRepoGroup === true;
        item.className = 'tree-item' + (isRootFolder || isRepoGroup ? ' root-folder expanded' : '');
        if (isRepoGroup) item.classList.add('repo-group');
        item.dataset.path = node.path || '';
        item.dataset.name = node.name.toLowerCase();

        if (isDir) {
            const displayName = node.alias || node.name;
            const iconSvg = isRepoGroup
                ? `<svg class="folder-icon" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/></svg>`
                : `<svg class="folder-icon" viewBox="0 0 24 24" fill="currentColor"><path d="M10 4H4a2 2 0 00-2 2v12a2 2 0 002 2h16a2 2 0 002-2V8a2 2 0 00-2-2h-8l-2-2z"/></svg>`;
            item.innerHTML = `
                <div class="tree-label">
                    ${iconSvg}
                    <span class="tree-name">${this.escapeHtml(displayName)}</span>
                    <svg class="arrow" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M8.59 16.59L13.17 12 8.59 7.41 10 6l6 6-6 6-1.41-1.41z"/>
                    </svg>
                </div>
                <div class="tree-children"></div>
            `;

            item.querySelector('.tree-label').addEventListener('click', () => {
                item.classList.toggle('expanded');
            });
        } else {
            item.innerHTML = `
                <div class="tree-label">
                    <svg class="file-icon" viewBox="0 0 24 24" fill="currentColor">
                        <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8l-6-6zm4 18H6V4h7v5h5v11z"/>
                    </svg>
                    <span class="tree-name">${this.escapeHtml(node.name)}</span>
                </div>
            `;

            item.querySelector('.tree-label').addEventListener('click', () => {
                this.loadFile(node.path);
            });
        }

        return item;
    }

    filterTree(query) {
        const items = document.querySelectorAll('.tree-item');
        const lowerQuery = query.toLowerCase();

        items.forEach(item => {
            if (!query) {
                item.style.display = '';
                return;
            }

            const name = item.dataset.name;
            const matches = name.includes(lowerQuery);
            item.style.display = matches ? '' : 'none';

            // Expand parent directories when searching
            if (matches) {
                let parent = item.parentElement;
                while (parent && parent.classList.contains('tree-children')) {
                    parent.parentElement.classList.add('expanded');
                    parent.parentElement.style.display = '';
                    parent = parent.parentElement.parentElement;
                }
            }
        });
    }

    // ========================================
    // File Loading & Rendering
    // ========================================
    async loadFile(path, updateHistory = true) {
        try {
            const response = await fetch(`/api/files/${encodeURIComponent(path)}`);
            if (!response.ok) throw new Error('Failed to load file');

            const data = await response.json();
            this.currentPath = path;

            // Update active state in tree
            document.querySelectorAll('.tree-label.active').forEach(el => {
                el.classList.remove('active');
            });
            const activeItem = document.querySelector(`.tree-item[data-path="${path}"] > .tree-label`);
            if (activeItem) {
                activeItem.classList.add('active');
                // Expand parent directories
                let parent = activeItem.parentElement.parentElement;
                while (parent && parent.classList.contains('tree-children')) {
                    parent.parentElement.classList.add('expanded');
                    parent = parent.parentElement.parentElement;
                }
            }

            // Render content
            this.renderContent(data);
            this.renderBreadcrumb(path, data.folderId);
            this.renderTOC(data.toc);

            // Update URL
            if (updateHistory) {
                window.history.pushState({ path }, data.title || path, `#${path}`);
            }

            // Scroll to top
            window.scrollTo(0, 0);

        } catch (error) {
            console.error('Error loading file:', error);
            this.showError('Failed to load file');
        }
    }

    renderContent(data) {
        const content = document.getElementById('content');
        content.innerHTML = `<div class="markdown-body">${data.html}</div>`;
    }

    renderBreadcrumb(path, folderId) {
        const breadcrumb = document.getElementById('breadcrumb');
        const parts = path.split('/');

        // Replace folder ID with alias if available
        if (this.folders.length > 0 && folderId !== undefined && folderId < this.folders.length) {
            parts[0] = this.folders[folderId].alias;
        }

        breadcrumb.innerHTML = parts.map((part, i) => {
            const isLast = i === parts.length - 1;
            return `
                <span class="breadcrumb-item${isLast ? '' : ''}">${this.escapeHtml(part)}</span>
                ${isLast ? '' : '<span class="breadcrumb-separator">/</span>'}
            `;
        }).join('');
    }

    renderTOC(toc) {
        const tocSidebar = document.getElementById('tocSidebar');
        const tocNav = document.getElementById('tocNav');

        if (!toc || toc.length <= 1) {
            tocSidebar.classList.remove('visible');
            return;
        }

        tocNav.innerHTML = toc.map(item => `
            <a href="#${item.anchor}"
               class="toc-link"
               data-level="${item.level}"
               data-anchor="${item.anchor}">
                ${this.escapeHtml(item.title)}
            </a>
        `).join('');

        tocSidebar.classList.add('visible');

        // Bind click events
        tocNav.querySelectorAll('.toc-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const anchor = link.dataset.anchor;
                const target = document.getElementById(anchor);
                if (target) {
                    target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }
            });
        });
    }

    updateActiveTocItem() {
        const headings = document.querySelectorAll('.markdown-body h1, .markdown-body h2, .markdown-body h3');
        const tocLinks = document.querySelectorAll('.toc-link');

        let current = null;
        headings.forEach(heading => {
            const rect = heading.getBoundingClientRect();
            if (rect.top <= 100) {
                current = heading.id;
            }
        });

        tocLinks.forEach(link => {
            link.classList.toggle('active', link.dataset.anchor === current);
        });
    }

    showError(message) {
        const content = document.getElementById('content');
        content.innerHTML = `
            <div class="welcome">
                <div class="welcome-icon" style="color: var(--error)">
                    <svg viewBox="0 0 24 24" width="80" height="80" fill="currentColor">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
                    </svg>
                </div>
                <h1>Error</h1>
                <p>${this.escapeHtml(message)}</p>
            </div>
        `;
    }

    // ========================================
    // WebSocket & Hot Reload
    // ========================================
    initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/ws`;

        try {
            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = () => {
                this.reconnectAttempts = 0;
                this.showConnectionStatus(true);
            };

            this.ws.onmessage = (event) => {
                const message = JSON.parse(event.data);
                this.handleWSMessage(message);
            };

            this.ws.onclose = () => {
                this.showConnectionStatus(false);
                this.scheduleReconnect();
            };

            this.ws.onerror = () => {
                this.showConnectionStatus(false);
            };
        } catch (error) {
            console.error('WebSocket connection failed:', error);
        }
    }

    handleWSMessage(message) {
        if (message.type === 'fileChange') {
            const { event, path } = message.payload;

            // Refresh tree on any change
            if (event === 'create' || event === 'remove') {
                this.loadFileTree();
            }

            // Reload current file if it was modified
            if (event === 'update' && this.currentPath === path) {
                this.loadFile(path, false);
            }
        }
    }

    scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.log('Max reconnection attempts reached');
            return;
        }

        const delay = Math.pow(2, this.reconnectAttempts) * 1000;
        this.reconnectAttempts++;

        setTimeout(() => {
            console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            this.initWebSocket();
        }, delay);
    }

    showConnectionStatus(connected) {
        const status = document.getElementById('connectionStatus');
        status.classList.toggle('visible', true);
        status.classList.toggle('disconnected', !connected);
        status.querySelector('.status-text').textContent = connected ? 'Connected' : 'Disconnected';

        if (connected) {
            setTimeout(() => {
                status.classList.remove('visible');
            }, 2000);
        }
    }

    // ========================================
    // URL Routing
    // ========================================
    handleInitialRoute() {
        const hash = window.location.hash.slice(1);
        if (hash) {
            this.loadFile(hash, false);
        }
    }

    // ========================================
    // Utilities
    // ========================================
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize application
document.addEventListener('DOMContentLoaded', () => {
    window.markhub = new MarkHub();
});
