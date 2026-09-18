//go:build !windows

package service

import "golang.org/x/sys/unix"

// diskUsage reports free and total bytes for the filesystem containing path.
func diskUsage(path string) (freeBytes, totalBytes uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	freeBytes = stat.Bavail * uint64(stat.Bsize)
	totalBytes = stat.Blocks * uint64(stat.Bsize)
	return freeBytes, totalBytes, nil
}
