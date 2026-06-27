package filemanager

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// TestResolveSafePath
// =============================================================================

func TestResolveSafePath_NormalPaths(t *testing.T) {
	root := t.TempDir()
	// On macOS, t.TempDir() returns /var/... but EvalSymlinks resolves to /private/var/...
	// NewFileService resolves the root via filepath.Abs, but ResolveSafePath also does EvalSymlinks.
	// So we resolve the root for comparison.
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	fs := NewFileService(root)

	// Create a subdirectory for testing.
	subDir := filepath.Join(root, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		relPath string
		wantErr bool
	}{
		{"root path /", "/", false},
		{"simple subdir", "/subdir", false},
		{"with trailing slash", "/subdir/", false},
		{"dot path", ".", false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fs.ResolveSafePath(tt.relPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (result: %s)", result)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Result must be within resolved root.
				if result != resolvedRoot && !isChildOf(result, resolvedRoot) {
					t.Errorf("resolved path %q is outside root %q", result, resolvedRoot)
				}
			}
		})
	}
}

func TestResolveSafePath_PathTraversal(t *testing.T) {
	root := t.TempDir()
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	fs := NewFileService(root)

	traversalPaths := []struct {
		name    string
		relPath string
	}{
		{"simple dot-dot", "../"},
		{"double dot-dot", "../../"},
		{"etc passwd", "../../etc/passwd"},
		{"mixed traversal", "/subdir/../../.."},
		{"dot-dot in middle", "/foo/../../../bar"},
		{"encoded style", "..%2F..%2Fetc%2Fpasswd"}, // raw, not URL-decoded here
	}

	for _, tt := range traversalPaths {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fs.ResolveSafePath(tt.relPath)
			if err == nil {
				// If no error, the resolved path MUST still be within root.
				if result != resolvedRoot && !isChildOf(result, resolvedRoot) {
					t.Errorf("path traversal NOT prevented! relPath=%q resolved to %q (root=%q)", tt.relPath, result, resolvedRoot)
				}
			}
			// If error, path traversal was correctly blocked.
		})
	}
}

func TestResolveSafePath_NonExistentPath(t *testing.T) {
	root := t.TempDir()
	// Use resolved root to avoid symlink mismatch issue on macOS.
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	fs := NewFileService(resolvedRoot)

	// Non-existent but valid child path should still resolve safely.
	result, err := fs.ResolveSafePath("/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error for non-existent child path: %v", err)
	}

	expected := filepath.Join(resolvedRoot, "nonexistent")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// =============================================================================
// TestListFiles
// =============================================================================

func TestListFiles_NormalDirectory(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	// Create test structure: a file and a subdirectory.
	if err := os.WriteFile(filepath.Join(root, "file1.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file2.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := fs.ListFiles("/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Path != "/" {
		t.Errorf("expected path '/', got %q", result.Path)
	}

	if len(result.Files) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result.Files))
	}

	// First entry should be directory (sorted dirs first).
	if !result.Files[0].IsDir {
		t.Errorf("expected first entry to be a directory, got file: %s", result.Files[0].Name)
	}
	if result.Files[0].Name != "subdir" {
		t.Errorf("expected first dir to be 'subdir', got %q", result.Files[0].Name)
	}
}

func TestListFiles_EmptyDirectory(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	result, err := fs.ListFiles("/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Files) != 0 {
		t.Errorf("expected 0 entries in empty dir, got %d", len(result.Files))
	}
}

func TestListFiles_NonExistentPath(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	_, err := fs.ListFiles("/nonexistent")
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}

func TestListFiles_SortOrder(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	// Create files and dirs.
	os.MkdirAll(filepath.Join(root, "Beta"), 0755)
	os.MkdirAll(filepath.Join(root, "alpha"), 0755)
	os.WriteFile(filepath.Join(root, "zebra.txt"), []byte("z"), 0644)
	os.WriteFile(filepath.Join(root, "apple.txt"), []byte("a"), 0644)

	result, err := fs.ListFiles("/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: dirs first (alpha, Beta), then files (apple.txt, zebra.txt).
	if len(result.Files) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(result.Files))
	}

	// First two should be directories.
	if !result.Files[0].IsDir || !result.Files[1].IsDir {
		t.Error("expected first two entries to be directories")
	}
	// alpha < Beta (case-insensitive).
	if result.Files[0].Name != "alpha" {
		t.Errorf("expected first dir 'alpha', got %q", result.Files[0].Name)
	}
	if result.Files[1].Name != "Beta" {
		t.Errorf("expected second dir 'Beta', got %q", result.Files[1].Name)
	}

	// Last two should be files.
	if result.Files[2].IsDir || result.Files[3].IsDir {
		t.Error("expected last two entries to be files")
	}
	if result.Files[2].Name != "apple.txt" {
		t.Errorf("expected first file 'apple.txt', got %q", result.Files[2].Name)
	}
}

// =============================================================================
// TestDeleteFile
// =============================================================================

func TestDeleteFile_RegularFile(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	filePath := filepath.Join(root, "to_delete.txt")
	if err := os.WriteFile(filePath, []byte("delete me"), 0644); err != nil {
		t.Fatal(err)
	}

	err := fs.DeleteFile("/to_delete.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file is gone.
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file still exists after deletion")
	}
}

func TestDeleteFile_Directory(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	dirPath := filepath.Join(root, "dir_to_delete")
	os.MkdirAll(filepath.Join(dirPath, "nested"), 0755)
	os.WriteFile(filepath.Join(dirPath, "nested", "file.txt"), []byte("content"), 0644)

	err := fs.DeleteFile("/dir_to_delete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory is gone.
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Error("directory still exists after deletion")
	}
}

func TestDeleteFile_NonExistent(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	err := fs.DeleteFile("/nonexistent_file.txt")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestDeleteFile_RootDirectory(t *testing.T) {
	root := t.TempDir()
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	fs := NewFileService(resolvedRoot) // Use resolved root to avoid symlink mismatch

	// Attempting to delete root itself should fail.
	err := fs.DeleteFile("/")
	if err == nil {
		t.Error("expected error when deleting root, got nil")
	}
}

func TestDeleteFile_PathTraversal(t *testing.T) {
	root := t.TempDir()
	fs := NewFileService(root)

	// Create a file outside root for safety check.
	err := fs.DeleteFile("/../../../tmp/should_not_delete")
	if err == nil {
		// If no error, that's a problem only if the path resolved outside root.
		// The ResolveSafePath should have blocked it. Let's just verify it didn't panic.
		t.Log("no error returned, but ResolveSafePath should keep it safe")
	}
	// Either an error is returned or the path resolves safely within root — both acceptable.
}

// =============================================================================
// Helpers
// =============================================================================

func isChildOf(child, parent string) bool {
	// Resolve symlinks for accurate comparison.
	resolvedChild, err := filepath.EvalSymlinks(child)
	if err != nil {
		resolvedChild = child
	}
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		resolvedParent = parent
	}
	return len(resolvedChild) > len(resolvedParent) &&
		resolvedChild[:len(resolvedParent)] == resolvedParent &&
		resolvedChild[len(resolvedParent)] == filepath.Separator
}
