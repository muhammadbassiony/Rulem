package fileops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

// ValidatePathSecurity applies rulem's policy to a path that does not exist
// yet, and therefore cannot be opened.
//
// # This is not containment
//
// Containment is [Dir]: a handle on an open directory, against which every name
// is resolved one component at a time. Where there is something to open, open
// it — a claim about a string is not a claim about the filesystem, and the gap
// between the two is where this package's historical bugs lived.
//
// This function exists for the one case where that is impossible: a path rulem
// is about to *create*. Its only callers prepare the destination of a `git
// clone` that has not happened yet, so there is no handle to hold and nothing
// to resolve against.
//
// # What it checks
//
//   - The path is not empty or whitespace-only.
//   - No component is "..". Note that [path/filepath.IsLocal] cannot stand in
//     here: it rejects every absolute path, and both callers pass one.
//   - An absolute path is not a reserved system directory ([IsReservedDirectory]).
//     This half is pure rulem policy — an opinion about where an application
//     should not put its data — and has no standard-library equivalent.
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

	if hasParentTraversal(path) {
		return fmt.Errorf("path traversal not allowed")
	}

	// Additional check for absolute paths that might be dangerous
	if filepath.IsAbs(path) && IsReservedDirectory(filepath.Clean(path)) {
		return fmt.Errorf("path traversal not allowed")
	}

	return nil
}

// hasParentTraversal reports whether any component of path is a ".." element.
//
// Matching on the ".." substring instead would also reject legitimate names
// such as "my..notes.md" or "v1..v2.md", which are perfectly valid rule files.
//
// Backslashes count as separators on every platform, not just Windows: a path
// like `..\..\etc` is traversal wherever it ends up being resolved, and a
// filename genuinely containing a backslash-delimited ".." component is not a
// case worth accommodating.
func hasParentTraversal(path string) bool {
	normalized := strings.ReplaceAll(filepath.ToSlash(path), `\`, "/")
	return slices.Contains(strings.Split(normalized, "/"), "..")
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

// SanitizeRelativePath validates and normalizes a user-supplied relative path that
// may contain forward-slash separated subdirectory segments (e.g. "backend/api-rules.md").
// It is intended for choosing a destination inside a trusted storage root while
// preventing the path from escaping that root.
//
// Behavior:
//   - A bare filename (no "/" separator) behaves exactly like SanitizeFilename, so
//     existing callers keep their current semantics.
//   - Otherwise the path is split on "/" and each segment is validated and sanitized
//     like an individual filename.
//
// Rules:
//   - Empty or whitespace-only paths are rejected.
//   - A leading "/" is not an absolute filesystem path here: the result is always
//     interpreted relative to a storage root, so "/backend/api.md" is accepted and
//     normalized to "backend/api.md" (root of the storage directory).
//   - Platform-absolute forms that survive that normalization (e.g. a Windows
//     volume path like "C:/rules.md") are rejected.
//   - Empty segments (e.g. "a//b.md" or a trailing "/") are rejected.
//   - "." and ".." segments (path traversal) are rejected.
//
// Returns:
//   - string: The cleaned relative path using the OS path separator, suitable for
//     joining onto a storage root.
//   - error: Validation error if the path is unsafe or invalid.
//
// Usage example:
//
//	rel, err := fileops.SanitizeRelativePath("backend/api-rules.md")
//	if err != nil {
//	    return err
//	}
//	dest := filepath.Join(storageDir, rel)
func SanitizeRelativePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// A leading "/" means "root of the storage directory", not the filesystem
	// root, so strip it rather than rejecting the path. The result is still
	// validated segment by segment below and joined onto the storage root.
	trimmed = strings.TrimLeft(trimmed, "/")
	if trimmed == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Anything still absolute after that (e.g. a Windows volume path) cannot be
	// contained inside the storage root.
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("path must be relative, not absolute: %q", path)
	}

	// Fast path: a bare filename with no separators must behave exactly like
	// SanitizeFilename so existing behavior is preserved.
	if !strings.Contains(trimmed, "/") {
		return SanitizeFilename(trimmed)
	}

	segments := strings.Split(trimmed, "/")
	cleaned := make([]string, 0, len(segments))
	for _, seg := range segments {
		if seg == "" {
			return "", fmt.Errorf("path contains empty segment: %q", path)
		}
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("path traversal not allowed: %q", path)
		}
		cleanSeg, err := SanitizeFilename(seg)
		if err != nil {
			return "", fmt.Errorf("invalid path segment %q: %w", seg, err)
		}
		cleaned = append(cleaned, cleanSeg)
	}

	// Every segment has already been rejected if it was "", "." or ".." and
	// sanitized individually, so the join cannot produce an escaping path.
	// The result is addressed through a Dir anyway, which would refuse one.
	return filepath.Join(cleaned...), nil
}

// ExpandPath expands a leading "~" to the user's home directory.
//
// Parameters:
//   - path: The path to expand, which may be "~" or start with "~/"
//
// Returns:
//   - string: The expanded path, or the original path if there is nothing to
//     expand or the home directory cannot be determined
//
// A bare "~" expands to the home directory itself. Only a leading tilde that
// forms its own path component is expanded, so a file legitimately named
// "~backup" is left alone.
//
// Usage example:
//
//	expanded := fileops.ExpandPath("~/Documents/file.txt")
//	// Returns something like "/home/user/Documents/file.txt"
func ExpandPath(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") &&
		!strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
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
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = filepath.Clean(resolved)
	}

	// Always treat root as reserved
	if absPath == "/" || absPath == "\\" || absPath == "C:\\" {
		return true
	}

	for _, reserved := range canonicalReservedDirectories() {
		if strings.EqualFold(absPath, reserved) {
			return true
		}
		if hasPathPrefix(absPath, reserved) && !isUserTempDirectory(absPath, reserved) {
			return true
		}
	}

	return false
}

// canonicalReservedDirectories returns the reserved directory list resolved to
// absolute, symlink-free paths.
//
// Canonicalizing costs a filepath.EvalSymlinks per entry and the set never
// changes during a run, so it is computed once instead of on every call.
// IsReservedDirectory sits underneath ValidateStoragePath and
// ValidatePathSecurity, which run on every path the user types.
// Both the literal and the symlink-resolved form of each entry are kept. A
// path that exists is matched in its resolved form ("/private/etc/passwd"),
// while one that does not exist cannot be resolved at all and is only ever
// seen as written ("/etc/nope/deeper") - so neither form alone covers both.
var canonicalReservedDirectories = sync.OnceValue(func() []string {
	raw := getReservedDirectories()
	canonical := make([]string, 0, len(raw)*2)
	for _, dir := range raw {
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		canonical = append(canonical, abs)

		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			if resolved = filepath.Clean(resolved); resolved != abs {
				canonical = append(canonical, resolved)
			}
		}
	}
	return canonical
})

// hasPathPrefix reports whether path lies beneath prefix, comparing whole path
// components so that "/etchosts" does not match the prefix "/etc".
//
// Both arguments must already be canonical. The old code compared the incoming
// canonical path against the *unresolved* reserved entry, which on macOS meant
// "/private/etc/passwd" was never matched by the "/etc" entry - it only got
// caught because "/private/etc" happened to be listed separately.
func hasPathPrefix(path, prefix string) bool {
	if len(path) <= len(prefix) {
		return false
	}
	if !strings.EqualFold(path[:len(prefix)], prefix) {
		return false
	}
	return os.IsPathSeparator(path[len(prefix)])
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

// isUserTempDirectory reports whether path sits inside a legitimate temporary
// directory that is itself nested strictly inside reservedDir.
//
// This is the allow-list exception inside IsReservedDirectory: the path is
// already known to be under reservedDir, and this function decides whether to
// wave it through anyway. It is the one place where matching too loosely
// *weakens* a guard instead of tightening it, so both halves are exact.
//
//   - The temp root must be a whole-component prefix of path (hasPathPrefix),
//     not a substring: the old bare strings.HasPrefix let "/tmpfoo" pass as
//     "/tmp", and the old Windows strings.Contains(path, `\temp\`) exempted any
//     reserved path with a directory named "temp" at any depth.
//   - The temp root must itself be strictly below reservedDir. os.TempDir reads
//     the user-controlled $TMPDIR, so without this a TMPDIR at or above a
//     reserved directory - TMPDIR=/var against /var/log - would exempt the
//     entire reserved tree. A temp root nested inside the reserved directory
//     only carves out that one hole, which is what the exception is for.
func isUserTempDirectory(path, reservedDir string) bool {
	cleanPath := filepath.Clean(path)

	for _, tempRoot := range userTempRoots() {
		if !hasPathPrefix(tempRoot, reservedDir) {
			continue
		}
		if strings.EqualFold(cleanPath, tempRoot) || hasPathPrefix(cleanPath, tempRoot) {
			return true
		}
	}

	return false
}

// userTempRoots returns the directories that legitimately hold per-user
// temporary files on this platform, canonicalized.
//
// os.TempDir covers the configured location on every OS, including Windows,
// where it reads %TMP%/%TEMP% under the user profile. The extra entries are the
// well-known defaults that remain valid even when $TMPDIR points somewhere else.
func userTempRoots() []string {
	roots := []string{filepath.Clean(os.TempDir())}

	switch runtime.GOOS {
	case "darwin":
		roots = append(roots, "/var/folders", "/private/var/folders")
	case "linux":
		roots = append(roots, "/tmp", "/var/tmp")
	}

	return roots
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
//   - Tests write permissions by creating a randomly named probe file
//   - Cleans up the probe file after verification
//   - Returns error if directory creation or write test fails
//
// Security: the probe is created with a random name and O_EXCL through an
// os.Root scoped to the directory, so it cannot be pre-created by another
// process to turn the probe into a write somewhere else.
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

	root, err := os.OpenRoot(expandedPath)
	if err != nil {
		return fmt.Errorf("cannot open directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	// Test write permissions
	name, probe, err := createTempFile(root)
	if err != nil {
		return fmt.Errorf("no write permission in directory: %w", err)
	}
	_ = probe.Close()

	// Clean up the probe. A failure here doesn't matter - the directory is usable.
	_ = root.Remove(name)

	return nil
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

// ValidateContentSecurity checks that text content is safe to pass through the
// application as text: it must not contain control characters that would let it
// forge terminal output, break out of a structured message, or truncate a
// string at a NUL byte.
//
// Parameters:
//   - content: The string content to validate
//
// Returns:
//   - error: Validation error if the content contains control characters
//
// The function rejects:
//   - NUL bytes
//   - Every other C0 control character except newline, carriage return and tab
//     (this includes ESC, so ANSI escape sequences are rejected)
//
// What this function deliberately does NOT do is scan for HTML and JavaScript
// injection markers such as "<script", "javascript:" or "eval(". Those checks
// were removed because they were a category error: they are an HTML-context
// defense applied to plain text that is never rendered as HTML here. They
// rejected legitimate documents - any document about web security discusses
// exactly those tokens - while providing no real protection, since any
// substring blocklist is trivially evaded. Escape content at the point where
// it enters an HTML context instead; that is the only place the context is
// known.
//
// Usage example:
//
//	if err := fileops.ValidateContentSecurity(userInput); err != nil {
//	    return fmt.Errorf("suspicious content detected: %w", err)
//	}
func ValidateContentSecurity(content string) error {
	for _, r := range content {
		if r == 0 {
			return fmt.Errorf("content contains null bytes")
		}
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("content contains control characters")
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

	// Build the result in a single pass, collapsing each run of separators as
	// it is encountered. The previous implementation chained ReplaceAll("  ",
	// " "), (" ", "_"), ("--", "_") and ("__", "_"), which cannot collapse a
	// run longer than two - "a     b" came out as "a__b" - and gave different
	// answers depending on which replacement happened to run first.
	var b strings.Builder
	b.Grow(len(identifier))
	var pending rune // the run of separators seen since the last kept character
	var pendingLen int

	for _, r := range identifier {
		switch {
		case isIdentifierRune(r):
			if pendingLen > 0 && b.Len() > 0 {
				// A lone '-' or '_' is meaningful and kept as written; a space,
				// or any run of two or more, normalizes to a single '_'.
				if pendingLen == 1 && pending != ' ' {
					b.WriteRune(pending)
				} else {
					b.WriteByte('_')
				}
			}
			pendingLen = 0
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			pending = r
			pendingLen++
		}
	}

	// Trim separators, then truncate, then trim again: truncating first can
	// cut mid-run and leave a trailing '_' behind. Every retained rune is
	// ASCII, so a byte-length limit cannot split a multi-byte character.
	result := strings.Trim(b.String(), "_-.")
	if maxLength > 0 && len(result) > maxLength {
		result = strings.TrimRight(result[:maxLength], "_-.")
	}

	if result == "" {
		return "", fmt.Errorf("identifier becomes empty after sanitization")
	}

	return result, nil
}

// isIdentifierRune reports whether r is kept verbatim by SanitizeIdentifier.
// Periods are content, not separators: "rules.v2" keeps its dot.
func isIdentifierRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '.'
}

// IsDirEmpty checks if a directory is empty.
// This function safely opens a directory and checks if it contains any entries.
//
// Parameters:
//   - path: Directory path to check
//
// Returns:
//   - bool: true if directory is empty, false otherwise
//   - error: File system access errors
func IsDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
