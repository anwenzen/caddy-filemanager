package filemanager

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// apiResponse represents the standard JSON response envelope for all API endpoints.
type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// serveRequest is the main HTTP router. It dispatches requests to the appropriate
// handler based on the URL path and HTTP method.
func (fm *FileManager) serveRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/api/files" && r.Method == http.MethodGet:
		fm.handleListFiles(w, r)
	case path == "/api/files" && r.Method == http.MethodDelete:
		fm.handleDelete(w, r)
	case path == "/api/disk" && r.Method == http.MethodGet:
		fm.handleDiskInfo(w, r)
	default:
		fm.handleStaticFiles(w, r)
	}
}

// handleListFiles handles GET /api/files?path=<relative_path>.
// It returns a JSON list of files and directories at the given path.
func (fm *FileManager) handleListFiles(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		relPath = "/"
	}

	result, err := fm.fileService.ListFiles(relPath)
	if err != nil {
		fm.logger.Warn("failed to list files",
			zap.String("path", relPath),
			zap.Error(err),
		)
		writeJSON(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, "ok", result)
}

// handleDelete handles DELETE /api/files?path=<relative_path>.
// It verifies the delete password (if configured) and removes the target file or directory.
func (fm *FileManager) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Password verification: if a delete password is configured, require it.
	if fm.DeletePassword != "" {
		pwd := r.Header.Get("X-Delete-Password")
		if pwd != fm.DeletePassword {
			writeJSON(w, http.StatusUnauthorized, "密码错误", nil)
			return
		}
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" || relPath == "/" {
		writeJSON(w, http.StatusBadRequest, "不能删除根目录", nil)
		return
	}

	err := fm.fileService.DeleteFile(relPath)
	if err != nil {
		fm.logger.Warn("failed to delete file",
			zap.String("path", relPath),
			zap.Error(err),
		)
		writeJSON(w, http.StatusInternalServerError, "删除失败", nil)
		return
	}

	fm.logger.Info("file deleted", zap.String("path", relPath))
	writeJSON(w, http.StatusOK, "删除成功", nil)
}

// handleDiskInfo handles GET /api/disk.
// It returns the disk usage information for the configured root path.
func (fm *FileManager) handleDiskInfo(w http.ResponseWriter, r *http.Request) {
	info, err := fm.diskService.GetDiskInfo()
	if err != nil {
		fm.logger.Warn("failed to get disk info", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, "获取磁盘信息失败", nil)
		return
	}

	// Include password_required field so the frontend knows whether to show password input.
	type diskInfoWithMeta struct {
		*DiskInfo
		PasswordRequired bool `json:"password_required"`
	}

	data := &diskInfoWithMeta{
		DiskInfo:         info,
		PasswordRequired: fm.DeletePassword != "",
	}

	writeJSON(w, http.StatusOK, "ok", data)
}

// handleStaticFiles serves the embedded frontend assets.
// Root path "/" is rewritten to serve index.html.
func (fm *FileManager) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Determine the file path within the embedded filesystem.
	path := r.URL.Path
	if path == "/" || path == "" {
		path = "frontend/index.html"
	} else {
		path = "frontend" + path
	}

	// Attempt to read the file from the embedded filesystem.
	content, err := fs.ReadFile(frontendFS, path)
	if err != nil {
		// If file not found, serve index.html for SPA fallback.
		content, err = fs.ReadFile(frontendFS, "frontend/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		path = "frontend/index.html"
	}

	// Set content type based on file extension.
	contentType := inferContentType(path)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// inferContentType returns the MIME type based on file extension.
func inferContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return ""
	}
}

// writeJSON writes a standard API JSON response with the given HTTP status code.
func writeJSON(w http.ResponseWriter, httpStatus int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)

	code := 0
	if httpStatus != http.StatusOK {
		code = httpStatus
	}

	resp := apiResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}
