/**
 * File browser logic for Zephyrus file access interface
 * Handles index decryption, directory traversal, and file downloads
 */

class FileVault {
    constructor(username, password) {
        this.username = username;
        this.password = password;
        this.index = null;
        this.currentPath = '';
        this.repoURL = `https://raw.githubusercontent.com/${username}/.zephyrus/master`;
    }

    /**
     * Fetch and decrypt the vault index
     */
    async loadIndex() {
        try {
            const indexUrl = `${this.repoURL}/.config/index`;
            const response = await fetch(indexUrl);

            if (!response.ok) {
                throw new Error(`Failed to fetch index (${response.status}). Check username or permissions.`);
            }

            const encryptedBuffer = await response.arrayBuffer();
            const decryptedBuffer = await CRYPTO.decryptWithPassword(encryptedBuffer, this.password);
            const jsonString = new TextDecoder().decode(decryptedBuffer);
            
            this.index = JSON.parse(jsonString);
            return this.index;
        } catch (error) {
            if (error instanceof SyntaxError) {
                throw new Error('Invalid password or corrupted index');
            }
            throw error;
        }
    }

    /**
     * Get the current directory contents
     */
    getCurrentDirectory() {
        if (!this.index) {
            console.error('Index not loaded');
            return [];
        }

        // Handle different possible index structures
        let filesObj = this.index.files || this.index.Index || this.index;
        
        // If it's an array, convert to object with paths as key
        if (Array.isArray(filesObj)) {
            const converted = {};
            for (const item of filesObj) {
                if (item.Path) {
                    converted[item.Path] = item;
                }
            }
            filesObj = converted;
        }

        if (!filesObj || typeof filesObj !== 'object') {
            console.error('Invalid index structure:', this.index);
            return [];
        }

        const items = [];
        const seen = new Set();

        // If we're in a subfolder, try to get contents from folder object
        if (this.currentPath) {
            const folder = filesObj[this.currentPath];
            if (folder && folder.contents) {
                console.log('Found folder contents for:', this.currentPath, folder.contents);
                // Files are stored in folder.contents
                for (const [fileName, fileEntry] of Object.entries(folder.contents)) {
                    if (fileEntry.type === 'folder') {
                        if (!seen.has(fileName)) {
                            seen.add(fileName);
                            items.push({
                                type: 'directory',
                                name: fileName,
                                path: this.currentPath + '/' + fileName
                            });
                        }
                    } else {
                        const realName = fileEntry.realName || fileEntry.RealName || fileEntry.StorageName || fileEntry.storage_name || fileEntry.real_name;
                        const fileKey = fileEntry.fileKey || fileEntry.FileKey || fileEntry.file_key;
                        
                        if (realName && fileKey) {
                            items.push({
                                type: 'file',
                                name: fileName,
                                path: this.currentPath + '/' + fileName,
                                realName: realName,
                                fileKey: fileKey,
                                size: fileEntry.Size || fileEntry.size
                            });
                        }
                    }
                }
                return items.sort((a, b) => {
                    if (a.type !== b.type) return a.type === 'directory' ? -1 : 1;
                    return a.name.localeCompare(b.name);
                });
            }
        }

        // Otherwise, list top-level items or items in current path prefix
        const pathPrefix = this.currentPath ? this.currentPath + '/' : '';

        for (const [vaultPath, fileEntry] of Object.entries(filesObj)) {
            if (!vaultPath.startsWith(pathPrefix)) continue;

            const relativePath = vaultPath.slice(pathPrefix.length);
            const parts = relativePath.split('/').filter(p => p);

            if (parts.length === 1) {
                // Handle folder type entries
                if (fileEntry.type === 'folder' || (fileEntry.contents && !fileEntry.realName)) {
                    items.push({
                        type: 'directory',
                        name: parts[0],
                        path: vaultPath
                    });
                    continue;
                }

                // File in current directory
                const realName = fileEntry.realName || fileEntry.RealName || fileEntry.StorageName || fileEntry.storage_name || fileEntry.real_name;
                const fileKey = fileEntry.fileKey || fileEntry.FileKey || fileEntry.file_key;
                
                if (!realName || !fileKey) {
                    console.warn('Skipping file with missing realName or fileKey:', vaultPath, fileEntry);
                    continue;
                }
                
                items.push({
                    type: 'file',
                    name: parts[0],
                    path: vaultPath,
                    realName: realName,
                    fileKey: fileKey,
                    size: fileEntry.Size || fileEntry.size
                });
            } else if (parts.length > 1) {
                // Directory reference
                const dirName = parts[0];
                if (!seen.has(dirName)) {
                    seen.add(dirName);
                    items.push({
                        type: 'directory',
                        name: dirName,
                        path: pathPrefix + dirName
                    });
                }
            }
        }

        // Sort: directories first, then files, both alphabetically
        return items.sort((a, b) => {
            if (a.type !== b.type) {
                return a.type === 'directory' ? -1 : 1;
            }
            return a.name.localeCompare(b.name);
        });
    }

    /**
     * Navigate to a directory
     */
    navigateToDirectory(dirPath) {
        this.currentPath = dirPath;
    }

    /**
     * Navigate to parent directory
     */
    goUp() {
        const parts = this.currentPath.split('/').filter(p => p);
        parts.pop();
        this.currentPath = parts.join('/');
    }

    /**
     * Download and decrypt a file
     */
    async downloadFile(fileEntry) {
        try {
            console.log('Starting download for:', fileEntry.name, 'realName:', fileEntry.realName);
            
            const fileUrl = `${this.repoURL}/${fileEntry.realName}`;
            console.log('Fetching from URL:', fileUrl);
            
            const response = await fetch(fileUrl);

            if (!response.ok) {
                throw new Error(`Failed to fetch file (${response.status}). File may not exist in vault.`);
            }

            const encryptedBuffer = await response.arrayBuffer();
            console.log('Encrypted file size:', encryptedBuffer.byteLength, 'bytes');
            
            // First, decrypt the file key using the vault password
            const encryptedKeyHex = fileEntry.fileKey;
            console.log('Encrypted file key (hex):', encryptedKeyHex.substring(0, 20) + '...');
            
            const encryptedKeyBuffer = CRYPTO.hexToBuffer(encryptedKeyHex);
            console.log('Encrypted key buffer size:', encryptedKeyBuffer.length, 'bytes');
            
            let fileKeyBuffer;
            try {
                fileKeyBuffer = await CRYPTO.decryptWithPassword(encryptedKeyBuffer, this.password);
                console.log('Decrypted file key size:', fileKeyBuffer.length, 'bytes');
            } catch (e) {
                throw new Error(`Failed to decrypt file key: ${e.message}`);
            }
            
            if (fileKeyBuffer.length !== 32) {
                throw new Error(`Invalid file key length after decryption: expected 32 bytes, got ${fileKeyBuffer.length}`);
            }
            
            // Now decrypt the file using the decrypted key
            const decryptedBuffer = await CRYPTO.decryptWithKey(encryptedBuffer, fileKeyBuffer);
            console.log('Decrypted file size:', decryptedBuffer.byteLength, 'bytes');

            return decryptedBuffer;
        } catch (error) {
            console.error('Download error:', error);
            throw new Error(`Failed to download file: ${error.message}`);
        }
    }

    /**
     * Get human-readable current path
     */
    getCurrentPathDisplay() {
        return this.currentPath || 'root';
    }

    /**
     * Get breadcrumb navigation
     */
    getBreadcrumbs() {
        const crumbs = [{ name: 'Root', path: '' }];
        if (this.currentPath) {
            const parts = this.currentPath.split('/');
            let currentPath = '';
            for (const part of parts) {
                currentPath = currentPath ? currentPath + '/' + part : part;
                crumbs.push({ name: part, path: currentPath });
            }
        }
        return crumbs;
    }
}

/**
 * UI Controller for the file browser
 */
class FileBrowserUI {
    constructor() {
        this.vault = null;
        this.isLoading = false;
        this.notificationTimeout = null;
        this.defaultTagline = 'Securely browse and download your encrypted files';
    }

    /**
     * Check if user parameter exists in URL, logout if missing
     */
    checkURLParameter() {
        const params = new URLSearchParams(window.location.search);
        const username = params.get('u');

        // If no ?u= in URL and we have a cached session, logout
        if (!username && sessionStorage.getItem('vault_username')) {
            this.logout();
        }
    }

    /**
     * Check URL parameters and initialize
     */
    initializeFromURL() {
        const params = new URLSearchParams(window.location.search);
        let username = params.get('u');

        // If no username in URL, check sessionStorage
        if (!username) {
            username = sessionStorage.getItem('vault_username');
            if (username) {
                // Update URL with cached username
                window.history.replaceState({}, '', `?u=${encodeURIComponent(username)}`);
            }
        }

        if (!username) {
            return false;
        }

        document.getElementById('usernameInput').value = username;
        return true;
    }

    /**
     * Authenticate and load vault
     */
    async authenticate() {
        const username = document.getElementById('usernameInput').value.trim();
        const password = document.getElementById('passwordInput').value;

        if (!username || !password) {
            this.showError('Please enter both username and password');
            return;
        }

        this.isLoading = true;
        this.updateAuthUI();

        try {
            this.vault = new FileVault(username, password);
            this.showInfo('Loading vault index...');
            const index = await this.vault.loadIndex();
            
            // Debug: Log the index structure
            console.log('Vault index loaded successfully:', index);
            console.log('Index keys:', Object.keys(index));
            if (index.files) console.log('Files object keys:', Object.keys(index.files).slice(0, 5));
            if (index.Index) console.log('Index property keys:', Object.keys(index.Index).slice(0, 5));
            
            // Store credentials in sessionStorage for persistence during session
            sessionStorage.setItem('vault_username', username);
            sessionStorage.setItem('vault_password', password);
            
            // Update URL with username parameter
            window.history.replaceState({}, '', `?u=${encodeURIComponent(username)}`);
            
            this.showSuccess('Vault loaded successfully!');
            document.getElementById('authSection').classList.add('hidden');
            document.getElementById('browserSection').classList.remove('hidden');
            
            this.updateBreadcrumb();
            this.renderCurrentDirectory();
        } catch (error) {
            console.error('Authentication error:', error);
            this.showError(`Authentication failed: ${error.message}`);
        } finally {
            this.isLoading = false;
            this.updateAuthUI();
        }
    }

    /**
     * Render current directory contents
     */
    renderCurrentDirectory() {
        const items = this.vault.getCurrentDirectory();
        const fileList = document.getElementById('fileList');
        fileList.innerHTML = '';

        if (items.length === 0) {
            fileList.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">📁</div>
                    <div class="empty-text">This directory is empty</div>
                    <div class="text-muted">Navigate to another folder or go back</div>
                </div>
            `;
            return;
        }

        for (const item of items) {
            const element = document.createElement('div');
            element.className = 'file-item';

            if (item.type === 'directory') {
                element.innerHTML = `
                    <div class="file-icon">📁</div>
                    <div class="file-info">
                        <div class="file-name">${this.escapeHtml(item.name)}</div>
                        <div class="file-path">${this.escapeHtml(item.path)}</div>
                    </div>
                `;
                element.addEventListener('click', () => {
                    this.vault.navigateToDirectory(item.path);
                    this.updateBreadcrumb();
                    this.renderCurrentDirectory();
                });
            } else {
                const sizeStr = item.size ? CRYPTO.formatBytes(item.size) : 'Unknown';
                element.innerHTML = `
                    <div class="file-icon">📄</div>
                    <div class="file-info">
                        <div class="file-name">${this.escapeHtml(item.name)}</div>
                        <div class="file-path">${this.escapeHtml(item.path)} • ${sizeStr}</div>
                    </div>
                    <div class="file-actions">
                        <button class="btn-secondary btn-small btn-download" onclick="fileBrowser.downloadAndShowFile('${this.escapeAttr(item.path)}')">
                            📥 Download
                        </button>
                    </div>
                `;
            }

            fileList.appendChild(element);
        }
    }

    /**
     * Update breadcrumb navigation
     */
    updateBreadcrumb() {
        const breadcrumbs = this.vault.getBreadcrumbs();
        const breadcrumbDiv = document.getElementById('breadcrumb');
        breadcrumbDiv.innerHTML = '';

        for (let i = 0; i < breadcrumbs.length; i++) {
            const crumb = breadcrumbs[i];
            const span = document.createElement('span');
            span.className = 'breadcrumb-item';
            span.textContent = crumb.name;
            span.onclick = () => {
                this.vault.currentPath = crumb.path;
                this.renderCurrentDirectory();
                this.updateBreadcrumb();
            };
            breadcrumbDiv.appendChild(span);

            if (i < breadcrumbs.length - 1) {
                const sep = document.createElement('span');
                sep.className = 'breadcrumb-separator';
                sep.textContent = ' / ';
                breadcrumbDiv.appendChild(sep);
            }
        }
    }

    /**
     * Download and display a file
     */
    async downloadAndShowFile(filePath) {
        const items = this.vault.getCurrentDirectory();
        const fileEntry = items.find(item => item.path === filePath);

        if (!fileEntry || fileEntry.type !== 'file') {
            this.showError('File not found');
            return;
        }

        try {
            this.showInfo(`Downloading ${this.escapeHtml(fileEntry.name)}...`);
            const decryptedBuffer = await this.vault.downloadFile(fileEntry);
            
            // Create and trigger download
            const mimeType = CRYPTO.getMimeType(fileEntry.name);
            const blob = new Blob([decryptedBuffer], { type: mimeType });
            const url = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = fileEntry.name;
            document.body.appendChild(link);
            link.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(link);

            this.showSuccess(`${this.escapeHtml(fileEntry.name)} downloaded successfully!`);
        } catch (error) {
            this.showError(`Download failed: ${error.message}`);
        }
    }

    /**
     * Logout and return to auth screen
     */
    logout() {
        this.vault = null;
        document.getElementById('passwordInput').value = '';
        document.getElementById('authSection').classList.remove('hidden');
        document.getElementById('browserSection').classList.add('hidden');
        this.clearMessages();
        
        // Clear session storage
        sessionStorage.removeItem('vault_username');
        sessionStorage.removeItem('vault_password');
    }

    /**
     * Restore session from sessionStorage if available
     */
    async restoreSession() {
        const username = sessionStorage.getItem('vault_username');
        const password = sessionStorage.getItem('vault_password');

        if (!username || !password) {
            return false;
        }

        try {
            this.vault = new FileVault(username, password);
            await this.vault.loadIndex();
            
            document.getElementById('usernameInput').value = username;
            document.getElementById('passwordInput').value = password;
            document.getElementById('authSection').classList.add('hidden');
            document.getElementById('browserSection').classList.remove('hidden');
            
            this.updateBreadcrumb();
            this.renderCurrentDirectory();
            return true;
        } catch (error) {
            // Session restoration failed, clear it
            sessionStorage.removeItem('vault_username');
            sessionStorage.removeItem('vault_password');
            return false;
        }
    }

    /**
     * Update authentication button state
     */
    updateAuthUI() {
        const btn = document.getElementById('authButton');
        btn.disabled = this.isLoading;
        btn.textContent = this.isLoading ? '🔄 Loading...' : '🔓 Unlock Vault';
    }

    /**
     * Show status messages
     */
    showError(message) {
        this.showMessage(message, 'error', '❌');
    }

    showSuccess(message) {
        this.showMessage(message, 'success', '✅');
    }

    showInfo(message) {
        this.showMessage(message, 'info', 'ℹ️');
    }

    showMessage(message, type, emoji) {
        const tagline = document.getElementById('statusTagline');
        
        // Clear existing timeout
        if (this.notificationTimeout) {
            clearTimeout(this.notificationTimeout);
        }
        
        // Update tagline with notification
        tagline.textContent = `${emoji} ${message}`;
        
        // Change color based on type
        tagline.style.color = {
            'error': '#ff8a80',
            'success': '#81c784',
            'info': '#64b5f6'
        }[type] || '#a0a0a0';
        
        // Reset after 4 seconds
        this.notificationTimeout = setTimeout(() => {
            tagline.textContent = this.defaultTagline;
            tagline.style.color = '#a0a0a0';
            this.notificationTimeout = null;
        }, 4000);
    }

    clearMessages() {
        if (this.notificationTimeout) {
            clearTimeout(this.notificationTimeout);
        }
        const tagline = document.getElementById('statusTagline');
        tagline.textContent = this.defaultTagline;
        tagline.style.color = '#a0a0a0';
    }

    /**
     * Utility: escape HTML
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Utility: escape attribute
     */
    escapeAttr(text) {
        return text.replace(/'/g, '&#39;').replace(/"/g, '&quot;');
    }
}

// Global instance
let fileBrowser;

// Initialize on page load
window.addEventListener('load', async () => {
    fileBrowser = new FileBrowserUI();
    
    // Setup event listeners
    document.getElementById('authButton').addEventListener('click', () => fileBrowser.authenticate());
    document.getElementById('passwordInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') fileBrowser.authenticate();
    });
    
    // Update URL live as username is typed
    document.getElementById('usernameInput').addEventListener('input', (e) => {
        const username = e.target.value.trim();
        if (username) {
            window.history.replaceState({}, '', `?u=${encodeURIComponent(username)}`);
        } else {
            window.history.replaceState({}, '', window.location.pathname);
        }
    });

    // Check if ?u= parameter exists, logout if missing
    fileBrowser.checkURLParameter();

    // First, try to restore session from sessionStorage
    const sessionRestored = await fileBrowser.restoreSession();
    
    if (!sessionRestored) {
        // If no cached session, check for URL username parameter
        fileBrowser.initializeFromURL();
    }
});
