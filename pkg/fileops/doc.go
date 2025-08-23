// Package fileops provides secure, atomic file operations for Go applications.
//
// This package implements file operations with security-first design principles,
// focusing on atomicity, path validation, and robust error handling. It is designed
// for use in applications requiring reliable file manipulation, such as configuration
// management or data persistence, while minimizing risks from partial writes or
// filesystem inconsistencies.
//
// Key features:
//   - Atomic file copies to prevent data corruption
//   - Safe directory creation with proper permissions
//   - Comprehensive error reporting with wrapped errors
//   - Cleanup on failure to maintain filesystem integrity
//
// All functions require absolute paths and do not perform path traversal checks;
// callers should validate paths before use. Permissions are set to standard values
// (0644 for files, 0755 for directories) and may need adjustment based on use case.
//
// Example usage:
//
//	err := fileops.AtomicCopy("/source/file.txt", "/dest/file.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
package fileops
