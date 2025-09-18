package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ValidatePathSecurity performs comprehensive security validation on a file path.
// This function checks for common path traversal attacks and dangerous path patterns.
//
// The function validates:
//   - Path traversal attempts using ".." sequences
//   - Empty or whitespace-only paths
//   - Paths that resolve outside expected boundaries after cleaning
//
// Parameters:
//   - path: The file path to validate
//
// Returns:
//   - error: Validation errors if the path is considered unsafe
//
// Security considerations:
//   - This function performs static analysis and does not access the filesystem
//   - Additional validation may be needed for specific use cases
//   - Symlink resolution should be performed separately if needed
//
// Usage example:
//
//	if err := fileops.ValidatePathSecurity("../../etc/passwd"); err != nil {
//	    log.Printf("Unsafe path detected: %v", err)
//	    return err
//	}
func ValidatePathSecurity(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal in raw input
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Clean and re-check for traversal
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Additional check for absolute paths that might be dangerous
	if filepath.IsAbs(path) {
		// Use comprehensive reserved directory checking
		if IsReservedDirectory(cleanPath) {
			return fmt.Errorf("path traversal not allowed")
		}
	}

	return nil
}

// ValidateCWDPath validates that a destination path is safe relative to current working directory.
// This function ensures the path is relative and doesn't attempt to escape the CWD.
//
// Parameters:
//   - destPath: Destination path to validate (should be relative)
//
// Returns:
//   - error: Validation errors if the path is unsafe
//
// The function checks:
//   - Path is not empty
//   - Path is relative (not absolute)
//   - Path doesn't contain traversal sequences
//   - Cleaned path doesn't escape current directory
//
// Usage example:
//
//	if err := fileops.ValidateCWDPath("subdir/file.txt"); err != nil {
//	    return fmt.Errorf("invalid destination path: %w", err)
//	}
func ValidateCWDPath(destPath string) error {
	if destPath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	// Path must be relative
	if filepath.IsAbs(destPath) {
		return fmt.Errorf("destination path must be relative to current working directory")
	}

	// Check for path traversal
	if strings.Contains(destPath, "..") {
		return fmt.Errorf("path traversal not allowed in destination path")
	}

	// Clean and validate the path
	cleanPath := filepath.Clean(destPath)
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed in destination path")
	}

	return nil
}

// ValidateFileInDirectory validates that a file path is within a specified base directory
// and that the file exists and is accessible. This function helps prevent directory
// traversal attacks and ensures file containment.
//
// Parameters:
//   - filePath: Full path to the file to validate
//   - baseDir: Base directory that should contain the file
//
// Returns:
//   - error: Validation errors if the file is outside the directory or inaccessible
//
// The function performs:
//   - Path resolution to absolute paths
//   - Containment verification using relative path calculation
//   - File existence and accessibility checks
//   - File type validation (ensures it's a regular file)
//   - Basic symlink security validation
//
// Usage example:
//
//	err := fileops.ValidateFileInDirectory("/storage/file.txt", "/storage")
//	if err != nil {
//	    return fmt.Errorf("file validation failed: %w", err)
//	}
func ValidateFileInDirectory(filePath, baseDir string) error {
	// Resolve absolute paths
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("cannot resolve file path: %w", err)
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("cannot resolve base directory: %w", err)
	}

	// Check if file is within base directory
	relPath, err := filepath.Rel(absBaseDir, absFilePath)
	if err != nil {
		return fmt.Errorf("cannot determine relative path: %w", err)
	}

	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("file is not within base directory")
	}

	// Check if file exists and is accessible
	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filepath.Base(filePath))
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Ensure it's a regular file
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}

	// Basic symlink validation - resolve and check final destination
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(absFilePath)
		if err != nil {
			return fmt.Errorf("cannot resolve symlink: %w", err)
		}

		// Ensure resolved path is still within base directory
		relResolved, err := filepath.Rel(absBaseDir, resolved)
		if err != nil {
			return fmt.Errorf("cannot determine resolved relative path: %w", err)
		}

		if strings.HasPrefix(relResolved, "..") {
			return fmt.Errorf("symlink resolves outside base directory")
		}
	}

	return nil
}

// SanitizeFilename sanitizes a filename by removing or replacing dangerous characters.
// This function helps ensure filenames are safe for filesystem operations.
//
// Parameters:
//   - filename: The filename to sanitize
//
// Returns:
//   - string: Sanitized filename
//   - error: Validation errors for completely invalid filenames
//
// The function:
//   - Removes path separators and traversal sequences
//   - Trims whitespace
//   - Validates against reserved names
//   - Ensures the filename is not empty after sanitization
//
// Usage example:
//
//	clean, err := fileops.SanitizeFilename("../../../etc/passwd")
//	if err != nil {
//	    return err
//	}
//	// clean will be "passwd" (safe to use)
func SanitizeFilename(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}

	// Remove any path components - use only the base name
	clean := filepath.Base(filename)

	// Additional cleaning: remove any remaining dangerous patterns
	clean = strings.ReplaceAll(clean, "..", "")
	clean = strings.TrimSpace(clean)

	// Check for reserved names
	if clean == "" || clean == "." || clean == ".." {
		return "", fmt.Errorf("invalid filename after sanitization: %q", filename)
	}

	// Check for path separators that might have survived
	// Note: On Windows, backslashes are path separators, but on Unix they're valid filename characters
	if strings.ContainsAny(clean, `/`) {
		return "", fmt.Errorf("filename contains path separators: %q", clean)
	}

	return clean, nil
}

// ValidateFileAccess checks if a file exists and is accessible with specified permissions.
// This function provides a way to verify file accessibility before performing operations.
//
// Parameters:
//   - filePath: Path to the file to check
//   - requireWrite: Whether write access is required
//
// Returns:
//   - error: Access validation errors
//
// The function checks:
//   - File existence
//   - Read permissions (always required)
//   - Write permissions (if requireWrite is true)
//   - File is not a directory
//
// Usage example:
//
//	// Check if file is readable
//	if err := fileops.ValidateFileAccess("/path/to/file.txt", false); err != nil {
//	    return fmt.Errorf("cannot read file: %w", err)
//	}
//
//	// Check if file is writable
//	if err := fileops.ValidateFileAccess("/path/to/file.txt", true); err != nil {
//	    return fmt.Errorf("cannot write to file: %w", err)
//	}
func ValidateFileAccess(filePath string, requireWrite bool) error {
	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Ensure it's not a directory
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Test read access
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("file is not readable: %w", err)
	}
	file.Close()

	// Test write access if required
	if requireWrite {
		file, err := os.OpenFile(filePath, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("file is not writable: %w", err)
		}
		file.Close()
	}

	return nil
}

// ExpandPath expands a path that starts with "~/" to the user's home directory.
// This is a utility function for handling user home directory shortcuts.
//
// Parameters:
//   - path: The path to expand, which may start with "~/"
//
// Returns:
//   - string: The expanded path, or the original path if it doesn't start with "~/"
//
// Usage example:
//
//	expanded := fileops.ExpandPath("~/Documents/file.txt")
//	// Returns something like "/home/user/Documents/file.txt"
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// IsReservedDirectory checks if the path is a system or reserved directory
// that should not be used for application data storage. This function helps
// prevent applications from accidentally writing to critical system locations.
//
// Parameters:
//   - path: The path to check
//
// Returns:
//   - bool: true if the path is reserved/dangerous, false otherwise
//
// The function checks:
//   - System directories (like /etc, /bin, C:\Windows, etc.)
//   - Critical user directories (like ~/.ssh, ~/.gnupg)
//   - Resolves symlinks to check final destinations
//   - Platform-specific reserved locations
//
// Usage example:
//
//	if fileops.IsReservedDirectory("/etc/passwd") {
//	    return fmt.Errorf("cannot use system directory")
//	}
func IsReservedDirectory(path string) bool {
	// Convert to absolute path for comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		return true // If we can't resolve it, treat as reserved
	}
	absPath = filepath.Clean(absPath)

	// Resolve any symlinks in the path for comparison
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err == nil {
		absPath = resolvedPath // Use resolved path if available
	}

	// Always treat root as reserved
	if absPath == "/" || absPath == "\\" || absPath == "C:\\" {
		return true
	}

	absPath = filepath.Clean(absPath)
	reservedDirs := getReservedDirectories()

	for _, reserved := range reservedDirs {
		// Canonicalize the reserved directory
		reservedAbs, err := filepath.Abs(reserved)
		if err != nil {
			continue
		}
		resolvedReserved, err := filepath.EvalSymlinks(reservedAbs)
		if err == nil {
			reservedAbs = filepath.Clean(resolvedReserved)
		} else {
			reservedAbs = filepath.Clean(reservedAbs)
		}

		// Exact match
		if strings.EqualFold(absPath, reservedAbs) {
			return true
		}

		// Child directory match - but with exceptions
		reservedPrefix := strings.ToLower(reserved) + string(os.PathSeparator)
		pathLower := strings.ToLower(absPath)

		if strings.HasPrefix(pathLower, reservedPrefix) {
			// Exception: Allow user temp directories
			if isUserTempDirectory(absPath) {
				continue
			}
			return true
		}
	}

	return false
}

// getReservedDirectories returns platform-specific reserved directories
func getReservedDirectories() []string {
	var reservedDirs []string

	switch runtime.GOOS {
	case "windows":
		reservedDirs = []string{
			"C:\\Windows",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			"C:\\System32",
			"C:\\ProgramData\\Microsoft", // More specific
		}

	case "darwin": // macOS
		reservedDirs = []string{
			"/System",
			"/usr/bin",
			"/usr/sbin",
			"/bin",
			"/sbin",
			"/etc",
			"/var/log",  // More specific - not all of /var
			"/var/db",   // More specific
			"/var/root", // More specific
			"/Library/System",
			"/Applications", // System apps
			"/private/etc",
		}

	default: // Linux and other Unix
		reservedDirs = []string{
			"/bin",
			"/sbin",
			"/usr/bin",
			"/usr/sbin",
			"/etc",
			"/boot",
			"/dev",
			"/proc",
			"/sys",
			"/var/log",   // More specific
			"/var/lib",   // More specific
			"/var/cache", // More specific
			"/root",
		}
	}

	// Add current user's system directories to avoid
	if home, err := os.UserHomeDir(); err == nil {
		// Avoid critical user directories
		systemUserDirs := []string{
			filepath.Join(home, ".ssh"),
			filepath.Join(home, ".gnupg"),
		}
		reservedDirs = append(reservedDirs, systemUserDirs...)
	}

	return reservedDirs
}

// isUserTempDirectory detects legitimate user temp directories
func isUserTempDirectory(path string) bool {
	// macOS: /var/folders/xx/yyyy/T/ are user temp dirs
	if runtime.GOOS == "darwin" {
		if strings.Contains(path, "/var/folders/") {
			return true
		}
	}

	// Linux: /tmp is usually safe, /var/tmp may be safe
	if runtime.GOOS == "linux" {
		if strings.HasPrefix(path, "/tmp/") || path == "/tmp" {
			return true
		}
	}

	// Windows: temp directories under user profile
	if runtime.GOOS == "windows" {
		if strings.Contains(strings.ToLower(path), "\\temp\\") ||
			strings.Contains(strings.ToLower(path), "\\tmp\\") {
			return true
		}
	}

	// Check if path is under system temp directory
	systemTemp := os.TempDir()
	cleanSystemTemp := filepath.Clean(systemTemp)
	cleanPath := filepath.Clean(path)

	if strings.HasPrefix(cleanPath, cleanSystemTemp) {
		return true
	}

	return false
}

// ValidateDirectoryWritable tests if a directory is writable by creating a test file.
// This function has side effects and should be called after path validation.
//
// Parameters:
//   - dirPath: The directory path to test for write permissions
//
// Returns:
//   - error: Write permission validation errors
//
// The function:
//   - Creates the directory if it doesn't exist
//   - Tests write permissions by creating a temporary test file
//   - Cleans up the test file after verification
//   - Returns error if directory creation or write test fails
//
// Usage example:
//
//	if err := fileops.ValidateDirectoryWritable("/path/to/dir"); err != nil {
//	    return fmt.Errorf("directory not writable: %w", err)
//	}
func ValidateDirectoryWritable(dirPath string) error {
	expandedPath := ExpandPath(strings.TrimSpace(dirPath))

	// Create directory if it doesn't exist
	if err := EnsureDirectoryExists(expandedPath); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	// Test write permissions
	testFile := filepath.Join(expandedPath, ".fileops-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("no write permission in directory: %w", err)
	}

	// Clean up test file
	if err := os.Remove(testFile); err != nil {
		// Log but don't fail - the directory is usable
		// Note: In a library, we can't log directly, so we just ignore cleanup failures
	}

	return nil
}

// ValidatePathInHome checks if a path is within the user's home directory
// and returns the relative path from home. This function helps ensure
// paths don't escape the user's home directory boundary.
//
// Parameters:
//   - targetPath: The path to validate against home directory containment
//
// Returns:
//   - string: Relative path from home directory if valid
//   - error: Validation errors if path is outside home or invalid
//
// The function:
//   - Resolves the user's home directory
//   - Cleans both home and target paths
//   - Calculates relative path from home to target
//   - Ensures the relative path doesn't escape home (no ".." prefix)
//
// Usage example:
//
//	relPath, err := fileops.ValidatePathInHome("/home/user/documents/file.txt")
//	if err != nil {
//	    return fmt.Errorf("path not in home: %w", err)
//	}
//	// relPath will be "documents/file.txt"
func ValidatePathInHome(targetPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Clean both paths to handle . and .. properly
	cleanHome := filepath.Clean(homeDir)
	cleanTarget := filepath.Clean(targetPath)

	// Check if target is within home
	relPath, err := filepath.Rel(cleanHome, cleanTarget)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// If relative path starts with .. then it's outside home
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path is outside home directory")
	}

	return relPath, nil
}

// ValidateStoragePath performs comprehensive validation for storage directory paths.
// This function combines multiple security and accessibility checks for directory paths
// intended for application data storage.
//
// Parameters:
//   - path: The storage directory path to validate
//
// Returns:
//   - error: Validation errors if the path is unsafe or unsuitable
//
// The function validates:
//   - Path is not empty or whitespace-only
//   - Basic path security (no traversal attempts)
//   - Path must be absolute or relative to home directory (~/)
//   - Symlink security (resolved paths don't point to reserved directories)
//   - Reserved directory protection (system directories are rejected)
//   - Parent directory accessibility
//
// Usage example:
//
//	if err := fileops.ValidateStoragePath("~/Documents/myapp"); err != nil {
//	    return fmt.Errorf("invalid storage path: %w", err)
//	}
func ValidateStoragePath(path string) error {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return fmt.Errorf("storage directory cannot be empty")
	}

	// Use basic path security validation
	if err := ValidatePathSecurity(trimmedPath); err != nil {
		return err
	}

	expandedPath := ExpandPath(trimmedPath)

	// Check if it's an absolute path or relative to home
	if !filepath.IsAbs(expandedPath) && !strings.HasPrefix(trimmedPath, "~/") {
		return fmt.Errorf("path must be absolute or relative to home directory (~)")
	}

	// Check for symlink security: ensure symlinks don't point to reserved directories
	if resolved, err := filepath.EvalSymlinks(expandedPath); err == nil {
		if IsReservedDirectory(resolved) {
			return fmt.Errorf("path resolves to reserved directory")
		}
	}

	// Check for reserved directories (after symlink checks)
	if IsReservedDirectory(expandedPath) {
		return fmt.Errorf("cannot use system or reserved directories")
	}

	// Check if parent directory exists and is accessible
	parentDir := filepath.Dir(expandedPath)
	if parentDir != "." {
		if _, err := os.Stat(parentDir); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("parent directory does not exist: %s", parentDir)
			}
			return fmt.Errorf("cannot access parent directory: %w", err)
		}
	}

	return nil
}

// ValidateContentSecurity checks for potentially malicious content in strings.
// This function detects common injection attacks, control characters, and other
// suspicious patterns that could be used for security exploits.
//
// Parameters:
//   - content: The string content to validate
//
// Returns:
//   - error: Validation error if suspicious content is detected
//
// The function checks for:
//   - Control characters (except newlines, carriage returns, and tabs)
//   - Null bytes
//   - Script injection patterns (script tags, javascript:, eval, etc.)
//   - Other potentially dangerous content patterns
//
// Usage example:
//
//	if err := fileops.ValidateContentSecurity(userInput); err != nil {
//	    return fmt.Errorf("suspicious content detected: %w", err)
//	}
func ValidateContentSecurity(content string) error {
	// Check for control characters (except newlines and tabs)
	for _, r := range content {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("content contains control characters")
		}
	}

	// Check for null bytes
	if strings.Contains(content, "\x00") {
		return fmt.Errorf("content contains null bytes")
	}

	// Check for script injection patterns
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"data:text/html",
		"eval(",
		"exec(",
		"onload=",
		"onerror=",
		"onclick=",
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerContent, pattern) {
			return fmt.Errorf("content contains potentially malicious pattern: %s", pattern)
		}
	}

	return nil
}

// SanitizeIdentifier sanitizes a string to be safe for use as an identifier.
// This function removes dangerous characters while preserving readability,
// making it suitable for tool names, variable names, or other identifiers.
//
// Parameters:
//   - identifier: The string to sanitize
//   - maxLength: Maximum allowed length (0 for no limit)
//
// Returns:
//   - string: Sanitized identifier
//   - error: Validation error if the identifier becomes empty after sanitization
//
// The function:
//   - Allows only alphanumeric characters, spaces, hyphens, underscores, and periods
//   - Normalizes multiple consecutive separators
//   - Trims leading/trailing separators
//   - Enforces length limits if specified
//
// Usage example:
//
//	clean, err := fileops.SanitizeIdentifier("my-tool@name#123", 50)
//	if err != nil {
//	    return "", err
//	}
//	// clean will be "my-toolname123"
func SanitizeIdentifier(identifier string, maxLength int) (string, error) {
	if strings.TrimSpace(identifier) == "" {
		return "", fmt.Errorf("identifier cannot be empty")
	}

	var cleanName strings.Builder

	for _, r := range identifier {
		// Allow alphanumeric, spaces, hyphens, underscores, and periods
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == ' ' || r == '-' || r == '_' || r == '.' {
			cleanName.WriteRune(r)
		}
	}

	result := strings.TrimSpace(cleanName.String())

	// Replace multiple consecutive spaces/separators with single underscore
	result = strings.ReplaceAll(result, "  ", " ")
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, "--", "_")
	result = strings.ReplaceAll(result, "__", "_")

	// Limit length if specified
	if maxLength > 0 && len(result) > maxLength {
		result = result[:maxLength]
	}

	// Trim leading/trailing separators
	result = strings.Trim(result, "_-.")

	if result == "" {
		return "", fmt.Errorf("identifier becomes empty after sanitization")
	}

	return result, nil
}

// ValidateFileSizeLimit checks if a file size is within acceptable limits.
// This function helps prevent memory exhaustion from very large files.
//
// Parameters:
//   - filePath: Path to the file to check
//   - maxSize: Maximum allowed file size in bytes
//
// Returns:
//   - error: Validation error if file exceeds size limit or cannot be accessed
//
// The function:
//   - Checks file existence and accessibility
//   - Compares file size against the specified limit
//   - Returns descriptive errors for different failure modes
//
// Usage example:
//
//	// Limit files to 10MB
//	if err := fileops.ValidateFileSizeLimit("/path/to/file.txt", 10*1024*1024); err != nil {
//	    return fmt.Errorf("file too large: %w", err)
//	}
func ValidateFileSizeLimit(filePath string, maxSize int64) error {
	if maxSize <= 0 {
		return fmt.Errorf("invalid size limit: %d", maxSize)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filepath.Base(filePath))
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	if fileInfo.Size() > maxSize {
		return fmt.Errorf("file size %d bytes exceeds limit %d bytes", fileInfo.Size(), maxSize)
	}

	return nil
}
