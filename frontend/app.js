/**
 * Caddy File Manager - Frontend Application
 *
 * A vanilla JavaScript SPA for browsing and managing files served by the
 * Caddy File Manager plugin. Uses hash-based routing for navigation.
 */

(function () {
    'use strict';

    // ==========================================================================
    // State
    // ==========================================================================

    /** Whether the server requires a password for deletion. */
    let passwordRequired = false;

    /** Current file path being viewed. */
    let currentPath = '/';

    /** Pending delete target info. */
    let pendingDelete = { name: '', path: '', isDir: false };

    // ==========================================================================
    // DOM Elements
    // ==========================================================================

    const diskText = document.getElementById('disk-text');
    const diskProgress = document.getElementById('disk-progress');
    const breadcrumb = document.getElementById('breadcrumb');
    const fileList = document.getElementById('file-list');
    const emptyState = document.getElementById('empty-state');
    const deleteModal = document.getElementById('delete-modal');
    const deleteMessage = document.getElementById('delete-message');
    const passwordField = document.getElementById('password-field');
    const deletePasswordInput = document.getElementById('delete-password');
    const btnCancel = document.getElementById('btn-cancel');
    const btnConfirmDelete = document.getElementById('btn-confirm-delete');
    const modalClose = document.getElementById('modal-close');
    const toastContainer = document.getElementById('toast-container');

    // ==========================================================================
    // Initialization
    // ==========================================================================

    /**
     * Initialize the application: load disk info, set up routing, and load
     * the initial file list based on the current URL hash.
     */
    function init() {
        loadDiskInfo();
        setupEventListeners();
        handleHashChange();
        window.addEventListener('hashchange', handleHashChange);
    }

    /**
     * Set up global event listeners for the delete modal.
     */
    function setupEventListeners() {
        btnCancel.addEventListener('click', hideDeleteModal);
        modalClose.addEventListener('click', hideDeleteModal);
        btnConfirmDelete.addEventListener('click', confirmDelete);

        // Close modal on overlay click.
        deleteModal.addEventListener('click', function (e) {
            if (e.target === deleteModal) {
                hideDeleteModal();
            }
        });

        // Close modal on Escape key.
        document.addEventListener('keydown', function (e) {
            if (e.key === 'Escape' && deleteModal.style.display !== 'none') {
                hideDeleteModal();
            }
        });
    }

    // ==========================================================================
    // Routing
    // ==========================================================================

    /**
     * Handle URL hash changes. Extracts the path from the hash and loads files.
     */
    function handleHashChange() {
        const hash = window.location.hash;
        currentPath = hash ? decodeURIComponent(hash.slice(1)) : '/';
        if (!currentPath.startsWith('/')) {
            currentPath = '/' + currentPath;
        }
        loadFiles(currentPath);
    }

    // ==========================================================================
    // API Calls
    // ==========================================================================

    /**
     * Load the file list for the given path from the API and render it.
     * @param {string} path - The relative directory path.
     */
    function loadFiles(path) {
        fileList.innerHTML = '<tr><td colspan="4" class="loading-text">加载中...</td></tr>';
        emptyState.style.display = 'none';

        fetch('/api/files?path=' + encodeURIComponent(path))
            .then(function (res) { return res.json(); })
            .then(function (resp) {
                if (resp.code !== 0) {
                    showToast(resp.message || '加载失败', 'error');
                    fileList.innerHTML = '';
                    return;
                }
                renderBreadcrumb(resp.data.path);
                renderFileList(resp.data.files);
            })
            .catch(function () {
                showToast('网络错误，无法加载文件列表', 'error');
                fileList.innerHTML = '';
            });
    }

    /**
     * Load disk information from the API and update the header bar.
     */
    function loadDiskInfo() {
        fetch('/api/disk')
            .then(function (res) { return res.json(); })
            .then(function (resp) {
                if (resp.code !== 0) {
                    diskText.textContent = '磁盘信息不可用';
                    return;
                }
                var data = resp.data;
                passwordRequired = data.password_required || false;

                var usedText = formatSize(data.used);
                var totalText = formatSize(data.total);
                var percent = data.used_percent.toFixed(1);
                diskText.textContent = usedText + ' / ' + totalText + ' (' + percent + '% 已使用)';
                diskProgress.style.width = percent + '%';

                // Mark as danger if usage > 90%.
                if (data.used_percent > 90) {
                    diskProgress.classList.add('danger');
                } else {
                    diskProgress.classList.remove('danger');
                }
            })
            .catch(function () {
                diskText.textContent = '磁盘信息不可用';
            });
    }

    /**
     * Send a delete request to the API.
     * @param {string} path - The relative path of the file/directory to delete.
     * @param {string} password - The delete password (may be empty).
     */
    function deleteFile(path, password) {
        var headers = { 'Content-Type': 'application/json' };
        if (password) {
            headers['X-Delete-Password'] = password;
        }

        fetch('/api/files?path=' + encodeURIComponent(path), {
            method: 'DELETE',
            headers: headers
        })
            .then(function (res) { return res.json(); })
            .then(function (resp) {
                if (resp.code !== 0) {
                    showToast(resp.message || '删除失败', 'error');
                    return;
                }
                showToast('删除成功', 'success');
                // Reload the current directory.
                loadFiles(currentPath);
                // Refresh disk info.
                loadDiskInfo();
            })
            .catch(function () {
                showToast('网络错误，删除失败', 'error');
            });
    }

    // ==========================================================================
    // Rendering
    // ==========================================================================

    /**
     * Render the file list table body.
     * @param {Array} files - Array of file info objects.
     */
    function renderFileList(files) {
        if (!files || files.length === 0) {
            fileList.innerHTML = '';
            emptyState.style.display = 'block';
            return;
        }

        emptyState.style.display = 'none';
        var html = '';

        for (var i = 0; i < files.length; i++) {
            var file = files[i];
            var icon = file.is_dir ? '📁' : '📄';
            var nameClass = file.is_dir ? 'file-name-link dir' : 'file-name-link';
            var sizeText = file.is_dir ? '-' : formatSize(file.size);
            var timeText = formatTime(file.mod_time);

            var linkHref, linkTarget;
            if (file.is_dir) {
                // Navigate into the directory.
                var dirPath = currentPath === '/' ? '/' + file.name : currentPath + '/' + file.name;
                linkHref = '#' + encodeURIComponent(dirPath);
                linkTarget = '';
            } else {
                linkHref = 'javascript:void(0)';
                linkTarget = '';
            }

            var filePath = currentPath === '/' ? '/' + file.name : currentPath + '/' + file.name;

            html += '<tr>';
            html += '<td><div class="file-name">';
            html += '<span class="file-name-icon">' + icon + '</span>';
            html += '<a href="' + linkHref + '" class="' + nameClass + '"' + linkTarget + '>' + escapeHtml(file.name) + '</a>';
            html += '</div></td>';
            html += '<td class="file-size">' + sizeText + '</td>';
            html += '<td class="file-time">' + timeText + '</td>';
            html += '<td style="text-align:center">';
            html += '<button class="btn btn-delete" onclick="window.__showDeleteModal(\'' + escapeJs(file.name) + '\', \'' + escapeJs(filePath) + '\', ' + file.is_dir + ')" title="删除">🗑️</button>';
            html += '</td>';
            html += '</tr>';
        }

        fileList.innerHTML = html;
    }

    /**
     * Render the breadcrumb navigation for the given path.
     * @param {string} path - The current directory path.
     */
    function renderBreadcrumb(path) {
        var parts = path.split('/').filter(function (p) { return p !== ''; });
        var html = '<a href="#/" class="breadcrumb-item">根目录</a>';

        var accumulated = '';
        for (var i = 0; i < parts.length; i++) {
            accumulated += '/' + parts[i];
            html += '<span class="breadcrumb-separator">/</span>';
            if (i === parts.length - 1) {
                html += '<span class="breadcrumb-item active">' + escapeHtml(parts[i]) + '</span>';
            } else {
                html += '<a href="#' + encodeURIComponent(accumulated) + '" class="breadcrumb-item">' + escapeHtml(parts[i]) + '</a>';
            }
        }

        breadcrumb.innerHTML = html;
    }

    // ==========================================================================
    // Delete Modal
    // ==========================================================================

    /**
     * Show the delete confirmation modal.
     * @param {string} fileName - Display name of the file.
     * @param {string} filePath - Relative path of the file.
     * @param {boolean} isDir - Whether the target is a directory.
     */
    function showDeleteModal(fileName, filePath, isDir) {
        pendingDelete = { name: fileName, path: filePath, isDir: isDir };

        var typeText = isDir ? '目录' : '文件';
        deleteMessage.textContent = '确定要删除' + typeText + ' "' + fileName + '" 吗？' +
            (isDir ? '（将递归删除所有内容）' : '');

        // Show password field if required.
        if (passwordRequired) {
            passwordField.style.display = 'block';
            deletePasswordInput.value = '';
        } else {
            passwordField.style.display = 'none';
        }

        deleteModal.style.display = 'flex';
        if (passwordRequired) {
            deletePasswordInput.focus();
        }
    }

    /**
     * Hide the delete confirmation modal.
     */
    function hideDeleteModal() {
        deleteModal.style.display = 'none';
        deletePasswordInput.value = '';
        pendingDelete = { name: '', path: '', isDir: false };
    }

    /**
     * Confirm deletion: collect password if needed and call the delete API.
     */
    function confirmDelete() {
        var password = '';
        if (passwordRequired) {
            password = deletePasswordInput.value.trim();
            if (!password) {
                showToast('请输入删除密码', 'info');
                deletePasswordInput.focus();
                return;
            }
        }

        // Capture the target path BEFORE hiding the modal, because
        // hideDeleteModal() resets pendingDelete (clearing the path).
        var pathToDelete = pendingDelete.path;
        hideDeleteModal();
        deleteFile(pathToDelete, password);
    }

    // Expose showDeleteModal globally for inline onclick handlers.
    window.__showDeleteModal = showDeleteModal;

    // ==========================================================================
    // Toast Notifications
    // ==========================================================================

    /**
     * Show a toast notification.
     * @param {string} msg - The message to display.
     * @param {string} type - One of 'success', 'error', 'info'.
     */
    function showToast(msg, type) {
        var toast = document.createElement('div');
        toast.className = 'toast ' + (type || 'info');
        toast.textContent = msg;
        toastContainer.appendChild(toast);

        // Remove the toast after animation completes.
        setTimeout(function () {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, 3000);
    }

    // ==========================================================================
    // Utilities
    // ==========================================================================

    /**
     * Format a byte count into a human-readable string (B, KB, MB, GB).
     * @param {number} bytes - The size in bytes.
     * @returns {string} Formatted size string.
     */
    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        var units = ['B', 'KB', 'MB', 'GB', 'TB'];
        var i = 0;
        var size = bytes;
        while (size >= 1024 && i < units.length - 1) {
            size /= 1024;
            i++;
        }
        return size.toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
    }

    /**
     * Format an ISO 8601 time string to a localized display format.
     * @param {string} isoString - ISO 8601 date-time string.
     * @returns {string} Formatted time string.
     */
    function formatTime(isoString) {
        if (!isoString) return '-';
        try {
            var date = new Date(isoString);
            if (isNaN(date.getTime())) return isoString;

            var year = date.getFullYear();
            var month = String(date.getMonth() + 1).padStart(2, '0');
            var day = String(date.getDate()).padStart(2, '0');
            var hour = String(date.getHours()).padStart(2, '0');
            var minute = String(date.getMinutes()).padStart(2, '0');

            return year + '-' + month + '-' + day + ' ' + hour + ':' + minute;
        } catch (e) {
            return isoString;
        }
    }

    /**
     * Escape HTML special characters to prevent XSS.
     * @param {string} str - The string to escape.
     * @returns {string} Escaped HTML string.
     */
    function escapeHtml(str) {
        var div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    /**
     * Escape a string for safe use inside a JavaScript string literal in HTML.
     * @param {string} str - The string to escape.
     * @returns {string} Escaped string safe for JS contexts.
     */
    function escapeJs(str) {
        return str.replace(/\\/g, '\\\\').replace(/'/g, "\\'").replace(/"/g, '\\"');
    }

    // ==========================================================================
    // Start Application
    // ==========================================================================

    // Wait for DOM to be ready, then initialize.
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
