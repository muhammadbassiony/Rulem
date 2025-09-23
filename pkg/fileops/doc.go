// Package fileops provides secure file operations with defense-in-depth validation patterns.
//
// This package implements atomic file operations combined with comprehensive security validations
// to prevent common attacks like path traversal, symlink exploits, and content injection.
//
// # Security Validation Patterns
//
// For maximum security, combine validation functions in this order:
//
// 1. **Path Security**: ValidatePathSecurity() - Prevents path traversal attacks
// 2. **File Size**: ValidateFileSizeLimit() - Prevents resource exhaustion
// 3. **File Access**: ValidateFileAccess() - Ensures file readability/writability
// 4. **Directory Containment**: ValidateFileInDirectory() - Prevents directory escapes
// 5. **Content Security**: ValidateContentSecurity() - Blocks malicious content
// 6. **Symlink Handling**: Check IsSymlink(), ValidateSymlinkSecurity() - Prevents symlink attacks
//
// # Example: Secure File Processing
//
//	// Comprehensive file validation before processing
//	if err := fileops.ValidatePathSecurity(filePath); err != nil {
//	    return fmt.Errorf("path security: %w", err)
//	}
//	if err := fileops.ValidateFileSizeLimit(filePath, 10*1024*1024); err != nil {
//	    return fmt.Errorf("file size: %w", err)
//	}
//	if err := fileops.ValidateFileAccess(filePath, false); err != nil {
//	    return fmt.Errorf("file access: %w", err)
//	}
//	if err := fileops.ValidateFileInDirectory(filePath, baseDir); err != nil {
//	    return fmt.Errorf("directory containment: %w", err)
//	}
//
//	content, _ := os.ReadFile(filePath)
//	if err := fileops.ValidateContentSecurity(string(content)); err != nil {
//	    return fmt.Errorf("content security: %w", err)
//	}
//
//	// Handle symlinks securely
//	if isLink, _ := fileops.IsSymlink(filePath); isLink {
//	    if err := fileops.ValidateSymlinkSecurity(filePath, []string{baseDir}); err != nil {
//	        return fmt.Errorf("symlink security: %w", err)
//	    }
//	}
//
// # Atomic Operations
//
// Use AtomicCopy() for reliable file transfers that prevent partial writes:
//
//	err := fileops.AtomicCopy(srcPath, destPath)
//	// Destination appears atomically or remains unchanged on failure
//
// # Directory Operations
//
// EnsureDirectoryExists() creates directories safely with proper permissions (0755).
package fileops
