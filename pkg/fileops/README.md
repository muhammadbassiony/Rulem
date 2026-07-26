# fileops Package

Secure file operations for Go applications with comprehensive validation patterns.

## Overview

The `fileops` package provides atomic file operations combined with defense-in-depth security validations. It prevents common security vulnerabilities like path traversal attacks, symlink exploits, control-character injection, and resource exhaustion.

## Security-First Design

All operations follow the principle of "defense in depth" - multiple validation layers ensure security even if one check fails. Functions are designed to fail securely by default.

## Usage Patterns

### 1. Comprehensive File Validation

For processing user-provided files, combine validations in this specific order for maximum security:

```go
func validateAndProcessFile(filePath, baseDir string, maxSize int64) error {
    // 1. Path security - prevent traversal attacks
    if err := fileops.ValidatePathSecurity(filePath); err != nil {
        return fmt.Errorf("path security validation failed: %w", err)
    }

    // 2. File size limits - prevent resource exhaustion
    if err := fileops.ValidateFileSizeLimit(filePath, maxSize); err != nil {
        return fmt.Errorf("file size validation failed: %w", err)
    }

    // 3. File access permissions - ensure readability
    if err := fileops.ValidateFileAccess(filePath, false); err != nil {
        return fmt.Errorf("file access validation failed: %w", err)
    }

    // 4. Directory containment - prevent directory escapes
    if err := fileops.ValidateFileInDirectory(filePath, baseDir); err != nil {
        return fmt.Errorf("directory containment validation failed: %w", err)
    }

    // 5. Read and validate content
    content, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }

    // 6. Content security - reject control characters
    if err := fileops.ValidateContentSecurity(string(content)); err != nil {
        return fmt.Errorf("content security validation failed: %w", err)
    }

    // 7. Symlink handling - prevent symlink-based attacks
    if isLink, err := fileops.IsSymlink(filePath); err != nil {
        return fmt.Errorf("symlink check failed: %w", err)
    } else if isLink {
        // Validate symlink points to safe locations
        allowedPaths := []string{baseDir}
        if err := fileops.ValidateSymlinkSecurity(filePath, allowedPaths); err != nil {
            return fmt.Errorf("symlink security validation failed: %w", err)
        }

        // Optionally resolve symlink for further processing
        resolvedPath, err := fileops.ResolveSymlink(filePath)
        if err != nil {
            return fmt.Errorf("symlink resolution failed: %w", err)
        }
        filePath = resolvedPath // Use resolved path for operations
    }

    // File is now safe to process
    return processFile(filePath, content)
}
```

This pattern is used in production code (see `internal/mcp/rulefile.go`) and provides comprehensive protection against:

- Path traversal attacks (`../../../etc/passwd`)
- Symlink-based directory escapes
- Control characters in file content (terminal escapes, NUL truncation)
- Resource exhaustion from large files
- Access to unauthorized files

### 2. Safe File Copy Operations

Use atomic operations to prevent data corruption:

```go
func safeFileCopy(srcPath, destPath string) error {
    // Validate source file
    if err := fileops.ValidateFileAccess(srcPath, false); err != nil {
        return fmt.Errorf("source validation failed: %w", err)
    }

    // Ensure destination directory exists
    destDir := filepath.Dir(destPath)
    if err := fileops.EnsureDirectoryExists(destDir); err != nil {
        return fmt.Errorf("destination directory creation failed: %w", err)
    }

    // Atomic copy - destination appears complete or unchanged
    if err := fileops.AtomicCopy(srcPath, destPath); err != nil {
        return fmt.Errorf("atomic copy failed: %w", err)
    }

    return nil
}
```

`AtomicCopy` writes through an `os.Root` scoped to the destination directory. The
temporary file is created with a random name and `O_EXCL`, so a symlink planted in
that directory cannot redirect the write, and concurrent copies to the same
destination don't collide. The destination inherits the source's permission bits.

### 3. Identifier Sanitization

When creating identifiers from user input:

```go
func createSafeIdentifier(userInput string) (string, error) {
    // Sanitize for security and consistency
    safeID, err := fileops.SanitizeIdentifier(userInput, 100)
    if err != nil {
        return "", fmt.Errorf("identifier sanitization failed: %w", err)
    }

    // Additional validation if needed
    if len(safeID) < 3 {
        return "", fmt.Errorf("identifier too short")
    }

    return safeID, nil
}
```

### 4. Directory Scanning with Security

For scanning directories while maintaining security boundaries:

```go
func scanDirectorySecurely(scanPath string) ([]string, error) {
    // Validate scan path security
    if err := fileops.ValidatePathSecurity(scanPath); err != nil {
        return nil, fmt.Errorf("scan path validation failed: %w", err)
    }

    // Create scanner with security options
    opts := &fileops.DirectoryScanOptions{
        MaxDepth:           10,  // Prevent deep recursion
        IncludeHidden:      false, // Skip hidden files for security
        SkipUnreadableDirs: true,  // Skip inaccessible directories
        ValidateFileAccess: true,  // Ensure files are accessible
    }

    scanner, err := fileops.NewDirectoryScanner(scanPath, opts)
    if err != nil {
        return nil, fmt.Errorf("scanner creation failed: %w", err)
    }
    defer scanner.Close()

    files, err := scanner.ScanDirectory()
    if err != nil {
        return nil, fmt.Errorf("directory scan failed: %w", err)
    }

    // Extract file paths
    var filePaths []string
    for _, file := range files {
        filePaths = append(filePaths, file.Path)
    }

    return filePaths, nil
}
```

### 5. Symlink Management

Safe symlink creation and validation:

```go
func createSafeSymlink(targetPath, linkPath string) error {
    // Validate target exists and is accessible
    if err := fileops.ValidateFileAccess(targetPath, false); err != nil {
        return fmt.Errorf("target validation failed: %w", err)
    }

    // Use relative symlinks for portability
    if err := fileops.CreateRelativeSymlink(targetPath, linkPath); err != nil {
        return fmt.Errorf("relative symlink creation failed: %w", err)
    }

    // Validate the created symlink
    allowedPaths := []string{filepath.Dir(targetPath)}
    if err := fileops.ValidateSymlinkSecurity(linkPath, allowedPaths); err != nil {
        // Clean up on validation failure. os.Remove deletes the link itself,
        // never the file it points at.
        _ = os.Remove(linkPath)
        return fmt.Errorf("symlink validation failed: %w", err)
    }

    return nil
}
```

## Security Considerations

### Path Validation Order

Always validate in this order to prevent bypasses:

1. **Path Security** first - catches obvious attacks
2. **File Size** second - prevents DoS before expensive operations
3. **Access Permissions** third - ensures file is readable
4. **Directory Containment** fourth - validates location
5. **Content Validation** last - checks actual content

### Symlink Attacks

Symlinks can bypass security checks. Always:

- Check if paths are symlinks using `IsSymlink()`
- Validate symlinks with `ValidateSymlinkSecurity()` before trusting them
- Resolve symlinks with `ResolveSymlink()` and re-validate the target

### Content Security

Use `ValidateContentSecurity()` on user-provided content to reject control
characters: NUL bytes, which truncate strings, and the rest of the C0 range
including ESC, which lets content forge terminal output.

It deliberately does **not** scan for `<script`, `javascript:` or `eval(`.
Those are an HTML-context defense, and markdown is never rendered as HTML
here - the check rejected legitimate rule files (any document about web
security quotes exactly those tokens) while providing no real protection,
since a substring blocklist is trivially evaded. Escape content where it
enters an HTML context; that is the only place the context is known.

### Resource Limits

Set appropriate limits:

- File sizes: 10MB for typical files, adjust based on use case
- Directory depths: 20-50 levels maximum
- Identifier lengths: 100-255 characters

## Error Handling

All functions return wrapped errors for better debugging. Use error wrapping:

```go
if err := fileops.ValidatePathSecurity(path); err != nil {
    return fmt.Errorf("security validation failed for %s: %w", path, err)
}
```

## Performance Notes

- Validation functions are designed to fail fast
- File access validation can be expensive - use selectively
- Directory scanning with `ValidateFileAccess: true` may be slow for large directories

## Testing

The package includes comprehensive tests covering security scenarios. Run tests with:

```bash
go test ./pkg/fileops -v
```

## References

- [OWASP File Upload Security](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html)
- [NIST Secure Software Development](https://csrc.nist.gov/Projects/ssdf)
- [Go Security Best Practices](https://github.com/securecodewarrior/essential-go-security)
