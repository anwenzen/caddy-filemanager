//go:build !windows

package filemanager

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// DiskService provides disk space information for the configured root path.
type DiskService struct {
	root string
}

// DiskInfo represents disk usage statistics.
type DiskInfo struct {
	Total       uint64  `json:"total"`        // Total disk space in bytes
	Free        uint64  `json:"free"`         // Free disk space in bytes
	Used        uint64  `json:"used"`         // Used disk space in bytes
	UsedPercent float64 `json:"used_percent"` // Used percentage (0-100)
}

// NewDiskService creates a new DiskService for the given root path.
func NewDiskService(root string) *DiskService {
	return &DiskService{root: root}
}

// GetDiskInfo retrieves the disk usage information for the filesystem
// containing the root path using the statfs system call.
func (ds *DiskService) GetDiskInfo() (*DiskInfo, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(ds.root, &stat)
	if err != nil {
		return nil, fmt.Errorf("无法获取磁盘信息")
	}

	// Calculate disk space values.
	// Total space = total blocks * block size
	total := stat.Blocks * uint64(stat.Bsize)
	// Free space = available blocks * block size (available to non-root users)
	free := stat.Bavail * uint64(stat.Bsize)
	// Used space = total - free
	used := total - free

	// Calculate used percentage, avoiding division by zero.
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return &DiskInfo{
		Total:       total,
		Free:        free,
		Used:        used,
		UsedPercent: usedPercent,
	}, nil
}
