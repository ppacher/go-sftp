package sftp

import (
	"os"
	"time"

	"github.com/nethack42/go-sftp/sshfxp"
)

// FileInfo holds file and directory information and implements os.FileInfo
type FileInfo struct {
	name      string
	size      int64
	mode      os.FileMode
	modtime   time.Time
	directory bool

	packet sshfxp.NameInfo
}

// Name returns the name of the directory or file
func (fi FileInfo) Name() string {
	return fi.name
}

// Size returns the file/directory size in bytes
func (fi FileInfo) Size() int64 {
	return fi.size
}

// Mode returns the access mode for the file or directory
// BUG: need to convert SFTP file modes to Golang os.FileMode
func (fi FileInfo) Mode() os.FileMode {
	return fi.mode
}

// ModTime returns the modification time of the file or directoy
func (fi FileInfo) ModTime() time.Time {
	return fi.modtime
}

// IsDir returns true if the file is actually a directory
// BUG: not yet implemented
func (fi FileInfo) IsDir() bool {
	return fi.directory
}

// Sys returns the underlying SSH_FXP packet (sshfxp.NameInfo)
func (fi FileInfo) Sys() interface{} {
	return fi.packet
}
