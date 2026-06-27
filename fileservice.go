package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileService handles all filesystem operations for the file manager,
// including listing files, deleting files, and validating paths.
type FileService struct {
	root string
}

// FileInfo represents metadata about a single file or directory.
type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"` // ISO 8601 format
	IsDir   bool   `json:"is_dir"`
}

// FileListResponse is the response structure for the list files API.
type FileListResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
}

// NewFileService creates a new FileService with the given root directory.
func NewFileService(root string) *FileService {
	// Resolve the root to an absolute path for consistent comparisons.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		absRoot = root
	}
	return &FileService{root: absRoot}
}

// ListFiles returns the list of files and directories at the given relative path.
// Directories are sorted before files, and each group is sorted alphabetically.
func (fs *FileService) ListFiles(relPath string) (*FileListResponse, error) {
	absPath, err := fs.ResolveSafePath(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取目录")
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			// Skip entries we can't stat.
			continue
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
			IsDir:   entry.IsDir(),
		})
	}

	// Sort: directories first, then files. Within each group, sort alphabetically.
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Normalize the path for the response.
	normalizedPath := filepath.Clean("/" + relPath)

	return &FileListResponse{
		Path:  normalizedPath,
		Files: files,
	}, nil
}

// DeleteFile removes the file or directory at the given relative path.
// Directories are removed recursively.
func (fs *FileService) DeleteFile(relPath string) error {
	absPath, err := fs.ResolveSafePath(relPath)
	if err != nil {
		return err
	}

	// Ensure the target exists before attempting deletion.
	_, err = os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("文件不存在")
		}
		return fmt.Errorf("无法访问文件")
	}

	// Prevent deletion of the root directory itself.
	if absPath == fs.root {
		return fmt.Errorf("不能删除根目录")
	}

	// Remove the file or directory recursively.
	err = os.RemoveAll(absPath)
	if err != nil {
		return fmt.Errorf("删除失败")
	}

	return nil
}

// ResolveSafePath validates and resolves a relative path to an absolute path,
// ensuring it stays within the root directory (path traversal protection).
func (fs *FileService) ResolveSafePath(relPath string) (string, error) {
	// Step 1: Clean the relative path, ensuring it starts with "/".
	cleaned := filepath.Clean("/" + relPath)

	// Step 2: Join with root to get the absolute path.
	absPath := filepath.Join(fs.root, cleaned)

	// Step 3: Resolve symlinks if the path exists.
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the path doesn't exist, EvalSymlinks will fail.
		// In that case, just use the cleaned absolute path for the prefix check.
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("路径无效")
		}
		resolved = absPath
	}

	// Step 4: Ensure the resolved path is within the root directory.
	// Also resolve the root's symlinks for accurate comparison.
	resolvedRoot, err := filepath.EvalSymlinks(fs.root)
	if err != nil {
		resolvedRoot = fs.root
	}

	// The resolved path must either equal the root or be a child of it.
	// We append os.PathSeparator to prevent matching "/root-other" against "/root".
	if resolved != resolvedRoot && !strings.HasPrefix(resolved, resolvedRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("路径越权访问")
	}

	return resolved, nil
}
