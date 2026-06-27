package filemanager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// newTestFileManager creates a FileManager instance suitable for testing.
// It sets up the root to a temp directory and initializes services.
func newTestFileManager(t *testing.T, root string, password string) *FileManager {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	fm := &FileManager{
		Root:           root,
		DeletePassword: password,
		logger:         logger,
		fileService:    NewFileService(root),
		diskService:    NewDiskService(root),
	}
	return fm
}

// parseAPIResponse unmarshals the response body into an apiResponse.
func parseAPIResponse(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v\nbody: %s", err, rec.Body.String())
	}
	return resp
}

// =============================================================================
// TestHandleListFiles
// =============================================================================

func TestHandleListFiles_Success(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "test.txt"), []byte("hello"), 0644)

	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/api/files?path=/", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	resp := parseAPIResponse(t, rec)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got %q", resp.Message)
	}

	// Verify data contains files.
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}
	files, ok := data["files"].([]interface{})
	if !ok {
		t.Fatalf("expected files to be an array, got %T", data["files"])
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestHandleListFiles_DefaultPath(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0644)

	fm := newTestFileManager(t, root, "")

	// No path parameter — should default to "/".
	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	resp := parseAPIResponse(t, rec)
	if resp.Code != 0 {
		t.Errorf("expected code 0 for default path, got %d", resp.Code)
	}
}

func TestHandleListFiles_InvalidPath(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/api/files?path=/nonexistent", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	resp := parseAPIResponse(t, rec)
	if resp.Code == 0 {
		t.Error("expected non-zero code for error response")
	}
}

func TestHandleListFiles_TraversalBlocked(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/api/files?path=../../etc", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	// Should either return error or resolve to root (both are safe).
	resp := parseAPIResponse(t, rec)
	if rec.Code == http.StatusOK {
		// If 200, verify the path didn't escape.
		data, ok := resp.Data.(map[string]interface{})
		if ok {
			path, _ := data["path"].(string)
			if path == "/etc" || path == "../../etc" {
				t.Error("path traversal not blocked in response")
			}
		}
	}
	// If not 200, traversal was blocked — pass.
}

// =============================================================================
// TestHandleDelete
// =============================================================================

func TestHandleDelete_NoPassword(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "victim.txt"), []byte("bye"), 0644)

	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodDelete, "/api/files?path=/victim.txt", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	resp := parseAPIResponse(t, rec)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}

	// Verify file is deleted.
	if _, err := os.Stat(filepath.Join(root, "victim.txt")); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestHandleDelete_WithCorrectPassword(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "secret.txt"), []byte("secret"), 0644)

	fm := newTestFileManager(t, root, "mypassword123")

	req := httptest.NewRequest(http.MethodDelete, "/api/files?path=/secret.txt", nil)
	req.Header.Set("X-Delete-Password", "mypassword123")
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	resp := parseAPIResponse(t, rec)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestHandleDelete_WithWrongPassword(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "safe.txt"), []byte("safe"), 0644)

	fm := newTestFileManager(t, root, "correctpass")

	req := httptest.NewRequest(http.MethodDelete, "/api/files?path=/safe.txt", nil)
	req.Header.Set("X-Delete-Password", "wrongpass")
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	// Verify file still exists.
	if _, err := os.Stat(filepath.Join(root, "safe.txt")); os.IsNotExist(err) {
		t.Error("file should NOT have been deleted with wrong password")
	}
}

func TestHandleDelete_MissingPasswordHeader(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "protected.txt"), []byte("protected"), 0644)

	fm := newTestFileManager(t, root, "secret123")

	req := httptest.NewRequest(http.MethodDelete, "/api/files?path=/protected.txt", nil)
	// No X-Delete-Password header set.
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	// Verify file still exists.
	if _, err := os.Stat(filepath.Join(root, "protected.txt")); os.IsNotExist(err) {
		t.Error("file should NOT have been deleted without password")
	}
}

func TestHandleDelete_RootPath(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodDelete, "/api/files?path=/", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for root deletion, got %d", rec.Code)
	}
}

func TestHandleDelete_EmptyPath(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodDelete, "/api/files?path=", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty path deletion, got %d", rec.Code)
	}
}

// =============================================================================
// TestHandleDiskInfo
// =============================================================================

func TestHandleDiskInfo_Success(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/api/disk", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", rec.Code, rec.Body.String())
	}

	resp := parseAPIResponse(t, rec)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}

	// Verify data has disk info fields.
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", resp.Data)
	}

	// Check required fields exist.
	requiredFields := []string{"total", "free", "used", "used_percent", "password_required"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("missing field %q in disk info response", field)
		}
	}

	// Total should be > 0.
	total, _ := data["total"].(float64)
	if total <= 0 {
		t.Errorf("expected total > 0, got %v", total)
	}
}

func TestHandleDiskInfo_PasswordRequiredFlag(t *testing.T) {
	root := t.TempDir()

	// Without password.
	fm1 := newTestFileManager(t, root, "")
	req := httptest.NewRequest(http.MethodGet, "/api/disk", nil)
	rec := httptest.NewRecorder()
	fm1.serveRequest(rec, req)

	resp1 := parseAPIResponse(t, rec)
	data1, _ := resp1.Data.(map[string]interface{})
	if pw, _ := data1["password_required"].(bool); pw != false {
		t.Errorf("expected password_required=false, got %v", pw)
	}

	// With password.
	fm2 := newTestFileManager(t, root, "pass123")
	req2 := httptest.NewRequest(http.MethodGet, "/api/disk", nil)
	rec2 := httptest.NewRecorder()
	fm2.serveRequest(rec2, req2)

	resp2 := parseAPIResponse(t, rec2)
	data2, _ := resp2.Data.(map[string]interface{})
	if pw, _ := data2["password_required"].(bool); pw != true {
		t.Errorf("expected password_required=true, got %v", pw)
	}
}

// =============================================================================
// TestServeStaticFile
// =============================================================================

func TestServeStaticFile_IndexHTML(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8', got %q", contentType)
	}

	// Body should contain some HTML.
	body := rec.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty body for index.html")
	}
}

func TestServeStaticFile_CSS(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/css; charset=utf-8', got %q", contentType)
	}
}

func TestServeStaticFile_JS(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/javascript; charset=utf-8" {
		t.Errorf("expected Content-Type 'application/javascript; charset=utf-8', got %q", contentType)
	}
}

func TestServeStaticFile_NotFound_FallbackToIndex(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/nonexistent-page", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	// SPA fallback: should return 200 with index.html content.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (SPA fallback), got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8' for SPA fallback, got %q", contentType)
	}
}

func TestServeStaticFile_CacheControl(t *testing.T) {
	root := t.TempDir()
	fm := newTestFileManager(t, root, "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	fm.serveRequest(rec, req)

	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "public, max-age=3600" {
		t.Errorf("expected Cache-Control 'public, max-age=3600', got %q", cacheControl)
	}
}

// =============================================================================
// TestWriteJSON
// =============================================================================

func TestWriteJSON_SuccessResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, "ok", map[string]string{"key": "value"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Code != 0 {
		t.Errorf("expected code 0 for 200 OK, got %d", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got %q", resp.Message)
	}
}

func TestWriteJSON_ErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusBadRequest, "bad request", nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp apiResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected code 400, got %d", resp.Code)
	}
}
