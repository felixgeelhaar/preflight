//go:build windows

package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Windows reparse point constants
const (
	// IO_REPARSE_TAG_MOUNT_POINT is the tag for junction points
	IO_REPARSE_TAG_MOUNT_POINT = 0xA0000003
	// IO_REPARSE_TAG_SYMLINK is the tag for symbolic links
	IO_REPARSE_TAG_SYMLINK = 0xA000000C
	// FSCTL_SET_REPARSE_POINT is the control code for setting reparse points
	FSCTL_SET_REPARSE_POINT = 0x000900A4
	// FSCTL_GET_REPARSE_POINT is the control code for getting reparse points
	FSCTL_GET_REPARSE_POINT = 0x000900A8
	// MAXIMUM_REPARSE_DATA_BUFFER_SIZE is the max size of reparse data
	MAXIMUM_REPARSE_DATA_BUFFER_SIZE = 16 * 1024
)

// REPARSE_DATA_BUFFER is the structure for mount point reparse data
type REPARSE_DATA_BUFFER struct {
	ReparseTag           uint32
	ReparseDataLength    uint16
	Reserved             uint16
	SubstituteNameOffset uint16
	SubstituteNameLength uint16
	PrintNameOffset      uint16
	PrintNameLength      uint16
	PathBuffer           [1]uint16
}

// IsJunction checks if a path is a junction point.
func (fs *RealFileSystem) IsJunction(path string) (bool, string) {
	// Get file attributes
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, ""
	}

	attrs, err := syscall.GetFileAttributes(pathPtr)
	if err != nil {
		return false, ""
	}

	// Check if it's a reparse point
	if attrs&syscall.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
		return false, ""
	}

	// Open the file to read the reparse data
	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return false, ""
	}
	defer syscall.CloseHandle(handle)

	// Read the reparse data
	buffer := make([]byte, MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var bytesReturned uint32
	err = syscall.DeviceIoControl(
		handle,
		FSCTL_GET_REPARSE_POINT,
		nil,
		0,
		&buffer[0],
		uint32(len(buffer)),
		&bytesReturned,
		nil,
	)
	if err != nil {
		return false, ""
	}

	// Parse the reparse data
	data := (*REPARSE_DATA_BUFFER)(unsafe.Pointer(&buffer[0]))
	if data.ReparseTag != IO_REPARSE_TAG_MOUNT_POINT {
		return false, ""
	}

	// Extract the target path
	// The path buffer starts after the fixed header fields
	nameOffset := data.SubstituteNameOffset / 2
	nameLength := data.SubstituteNameLength / 2

	// Calculate pointer to path buffer (after the header)
	headerSize := unsafe.Sizeof(REPARSE_DATA_BUFFER{}) - unsafe.Sizeof([1]uint16{})
	pathBufferStart := uintptr(unsafe.Pointer(&buffer[0])) + headerSize
	pathBuffer := (*[1024]uint16)(unsafe.Pointer(pathBufferStart))

	target := syscall.UTF16ToString(pathBuffer[nameOffset : nameOffset+nameLength])

	// Remove the \??\ prefix if present
	if len(target) > 4 && target[:4] == `\??\` {
		target = target[4:]
	}

	return true, target
}

// CreateJunction creates a junction point.
func (fs *RealFileSystem) CreateJunction(target, link string) error {
	// Ensure target is an absolute path
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for junction target %q: %w", target, err)
	}

	// Create the directory for the junction
	if err := os.MkdirAll(link, 0755); err != nil {
		return fmt.Errorf("failed to create junction directory %q: %w", link, err)
	}

	// Convert to UTF16
	linkPtr, err := syscall.UTF16PtrFromString(link)
	if err != nil {
		return fmt.Errorf("failed to convert junction link path %q to UTF16: %w", link, err)
	}

	// Open the directory
	handle, err := syscall.CreateFile(
		linkPtr,
		syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to open junction directory %q: %w", link, err)
	}
	defer syscall.CloseHandle(handle)

	// Prepare the target path with \??\ prefix
	targetWithPrefix := `\??\` + absTarget
	targetUTF16, err := syscall.UTF16FromString(targetWithPrefix)
	if err != nil {
		return fmt.Errorf("failed to convert junction target path %q to UTF16: %w", target, err)
	}

	// Calculate buffer size
	targetLen := len(targetUTF16) * 2
	bufferSize := 8 + 4 + 4 + targetLen + 2 // header + offsets + path + null

	buffer := make([]byte, bufferSize)

	// Fill in the reparse data buffer
	rdb := (*REPARSE_DATA_BUFFER)(unsafe.Pointer(&buffer[0]))
	rdb.ReparseTag = IO_REPARSE_TAG_MOUNT_POINT
	rdb.ReparseDataLength = uint16(bufferSize - 8)
	rdb.SubstituteNameOffset = 0
	rdb.SubstituteNameLength = uint16(targetLen - 2) // exclude null terminator
	rdb.PrintNameOffset = uint16(targetLen)
	rdb.PrintNameLength = 0

	// Copy target path
	headerSize := int(unsafe.Sizeof(REPARSE_DATA_BUFFER{})) - 2 // minus PathBuffer size
	copy(buffer[headerSize:], (*[1024]byte)(unsafe.Pointer(&targetUTF16[0]))[:targetLen])

	// Set the reparse point
	var bytesReturned uint32
	if err := syscall.DeviceIoControl(
		handle,
		FSCTL_SET_REPARSE_POINT,
		&buffer[0],
		uint32(len(buffer)),
		nil,
		0,
		&bytesReturned,
		nil,
	); err != nil {
		return fmt.Errorf("failed to create junction %q -> %q: %w", link, target, err)
	}
	return nil
}

// CreateLink creates the appropriate link type based on the target.
// On Windows: junction for directories (no admin required), symlink for files.
func (fs *RealFileSystem) CreateLink(target, link string) error {
	// Check if target is a directory
	info, err := os.Stat(target)
	if err != nil {
		// If target doesn't exist, try symlink (it may be created later)
		if symErr := os.Symlink(target, link); symErr != nil {
			return fmt.Errorf("failed to create symlink %q -> %q: %w", link, target, symErr)
		}
		return nil
	}

	if info.IsDir() {
		// Use junction for directories (no admin privileges required)
		return fs.CreateJunction(target, link)
	}

	// Use symlink for files (may require admin privileges)
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("failed to create symlink %q -> %q: %w", link, target, err)
	}
	return nil
}
