package fileops

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Test helpers for symlink operations

func createTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation failed on Windows: %v", err)
		}
		t.Fatalf("failed to create symlink: %v", err)
	}
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

// Tests for IsSymlink

func TestIsSymlink(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create test file and directory
	testFile := createTestFile(t, tempDir, "regular.txt", "content")
	testDir := filepath.Join(tempDir, "testdir")
	os.Mkdir(testDir, 0755)

	t.Run("regular file is not symlink", func(t *testing.T) {
		isLink, err := IsSymlink(testFile)
		if err != nil {
			t.Fatalf("IsSymlink failed: %v", err)
		}
		if isLink {
			t.Error("Regular file incorrectly identified as symlink")
		}
	})

	t.Run("directory is not symlink", func(t *testing.T) {
		isLink, err := IsSymlink(testDir)
		if err != nil {
			t.Fatalf("IsSymlink failed: %v", err)
		}
		if isLink {
			t.Error("Directory incorrectly identified as symlink")
		}
	})

	t.Run("symlink to file", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "file_link")
		createTestSymlink(t, testFile, linkPath)

		isLink, err := IsSymlink(linkPath)
		if err != nil {
			t.Fatalf("IsSymlink failed: %v", err)
		}
		if !isLink {
			t.Error("File symlink not identified correctly")
		}
	})

	t.Run("symlink to directory", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "dir_link")
		createTestSymlink(t, testDir, linkPath)

		isLink, err := IsSymlink(linkPath)
		if err != nil {
			t.Fatalf("IsSymlink failed: %v", err)
		}
		if !isLink {
			t.Error("Directory symlink not identified correctly")
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent")

		_, err := IsSymlink(nonExistentPath)
		if err == nil {
			t.Error("Expected error for non-existent path")
		}
	})
}

// Tests for CreateRelativeSymlink

func TestCreateRelativeSymlink(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create target file
	targetFile := createTestFile(t, tempDir, "target.txt", "target content")

	t.Run("create relative symlink", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "relative_link.txt")

		err := CreateRelativeSymlink(targetFile, linkPath)
		if err != nil {
			t.Fatalf("CreateRelativeSymlink failed: %v", err)
		}

		// Verify symlink exists
		isLink, err := IsSymlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to check symlink: %v", err)
		}
		if !isLink {
			t.Error("Created path is not a symlink")
		}

		// Verify symlink target is relative
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}
		if filepath.IsAbs(target) {
			t.Errorf("Expected relative target, got absolute: %s", target)
		}

		// Verify content is accessible through symlink
		content := readFileContent(t, linkPath)
		if content != "target content" {
			t.Errorf("Content mismatch through symlink. Expected %q, got %q", "target content", content)
		}
	})

	t.Run("create symlink in subdirectory", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "subdir")
		linkPath := filepath.Join(subDir, "nested_link.txt")

		err := CreateRelativeSymlink(targetFile, linkPath)
		if err != nil {
			t.Fatalf("CreateRelativeSymlink in subdirectory failed: %v", err)
		}

		// Verify content is accessible
		content := readFileContent(t, linkPath)
		if content != "target content" {
			t.Error("Content not accessible through nested symlink")
		}

		// Verify symlink resolves correctly
		resolved, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			t.Fatalf("Failed to resolve symlink: %v", err)
		}

		targetAbs, _ := filepath.Abs(targetFile)
		resolvedAbs, _ := filepath.Abs(resolved)

		// Handle macOS /private path differences
		targetCanonical, _ := filepath.EvalSymlinks(targetAbs)
		resolvedCanonical, _ := filepath.EvalSymlinks(resolvedAbs)

		if resolvedCanonical != targetCanonical {
			t.Errorf("Symlink doesn't resolve to target. Expected %s, got %s", targetCanonical, resolvedCanonical)
		}
	})

	t.Run("target does not exist", func(t *testing.T) {
		nonExistentTarget := filepath.Join(tempDir, "nonexistent.txt")
		linkPath := filepath.Join(tempDir, "broken_link.txt")

		err := CreateRelativeSymlink(nonExistentTarget, linkPath)
		if err == nil {
			t.Error("Expected error for non-existent target")
		}
		if !strings.Contains(err.Error(), "target does not exist") {
			t.Errorf("Expected 'target does not exist' error, got: %v", err)
		}
	})
}

// Tests for CreateAbsoluteSymlink

func TestCreateAbsoluteSymlink(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	targetFile := createTestFile(t, tempDir, "abs_target.txt", "absolute target content")

	t.Run("create absolute symlink", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "absolute_link.txt")

		err := CreateAbsoluteSymlink(targetFile, linkPath)
		if err != nil {
			t.Fatalf("CreateAbsoluteSymlink failed: %v", err)
		}

		// Verify symlink exists
		isLink, err := IsSymlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to check symlink: %v", err)
		}
		if !isLink {
			t.Error("Created path is not a symlink")
		}

		// Verify symlink target is absolute
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}
		if !filepath.IsAbs(target) {
			t.Errorf("Expected absolute target, got relative: %s", target)
		}

		// Verify content is accessible
		content := readFileContent(t, linkPath)
		if content != "absolute target content" {
			t.Error("Content not accessible through absolute symlink")
		}
	})

	t.Run("target does not exist", func(t *testing.T) {
		nonExistentTarget := filepath.Join(tempDir, "nonexistent_abs.txt")
		linkPath := filepath.Join(tempDir, "broken_abs_link.txt")

		err := CreateAbsoluteSymlink(nonExistentTarget, linkPath)
		if err == nil {
			t.Error("Expected error for non-existent target")
		}
	})
}

// Tests for ResolveSymlink

func TestResolveSymlink(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	targetFile := createTestFile(t, tempDir, "resolve_target.txt", "resolve content")

	t.Run("resolve single symlink", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "resolve_link.txt")
		createTestSymlink(t, targetFile, linkPath)

		resolved, err := ResolveSymlink(linkPath)
		if err != nil {
			t.Fatalf("ResolveSymlink failed: %v", err)
		}

		targetAbs, _ := filepath.Abs(targetFile)
		resolvedAbs, _ := filepath.Abs(resolved)

		// Handle macOS /private path differences
		targetCanonical, _ := filepath.EvalSymlinks(targetAbs)
		resolvedCanonical, _ := filepath.EvalSymlinks(resolvedAbs)

		if resolvedCanonical != targetCanonical {
			t.Errorf("Symlink resolution incorrect. Expected %s, got %s", targetCanonical, resolvedCanonical)
		}
	})

	t.Run("resolve symlink chain", func(t *testing.T) {
		link1 := filepath.Join(tempDir, "link1.txt")
		link2 := filepath.Join(tempDir, "link2.txt")

		// Create chain: link2 -> link1 -> targetFile
		createTestSymlink(t, targetFile, link1)
		createTestSymlink(t, link1, link2)

		resolved, err := ResolveSymlink(link2)
		if err != nil {
			t.Fatalf("ResolveSymlink chain failed: %v", err)
		}

		targetAbs, _ := filepath.Abs(targetFile)
		resolvedAbs, _ := filepath.Abs(resolved)

		// Handle macOS /private path differences
		targetCanonical, _ := filepath.EvalSymlinks(targetAbs)
		resolvedCanonical, _ := filepath.EvalSymlinks(resolvedAbs)

		if resolvedCanonical != targetCanonical {
			t.Errorf("Symlink chain resolution incorrect. Expected %s, got %s", targetCanonical, resolvedCanonical)
		}
	})

	t.Run("broken symlink", func(t *testing.T) {
		brokenTarget := filepath.Join(tempDir, "broken_target.txt")
		brokenLink := filepath.Join(tempDir, "broken_link.txt")
		createTestSymlink(t, brokenTarget, brokenLink)

		_, err := ResolveSymlink(brokenLink)
		if err == nil {
			t.Error("Expected error for broken symlink")
		}
	})

	t.Run("regular file", func(t *testing.T) {
		regularFile := createTestFile(t, tempDir, "regular_resolve.txt", "content")

		resolved, err := ResolveSymlink(regularFile)
		if err != nil {
			t.Fatalf("ResolveSymlink on regular file failed: %v", err)
		}

		regularAbs, _ := filepath.Abs(regularFile)
		resolvedAbs, _ := filepath.Abs(resolved)

		// Handle macOS /private path differences
		regularCanonical, _ := filepath.EvalSymlinks(regularAbs)
		resolvedCanonical, _ := filepath.EvalSymlinks(resolvedAbs)

		if resolvedCanonical != regularCanonical {
			t.Errorf("Regular file resolution incorrect. Expected %s, got %s", regularCanonical, resolvedCanonical)
		}
	})
}

// Tests for ValidateSymlinkSecurity

func TestValidateSymlinkSecurity(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create allowed directories
	allowedDir1 := filepath.Join(tempDir, "allowed1")
	allowedDir2 := filepath.Join(tempDir, "allowed2")
	os.MkdirAll(allowedDir1, 0755)
	os.MkdirAll(allowedDir2, 0755)

	// Create forbidden directory
	forbiddenDir := filepath.Join(tempDir, "forbidden")
	os.MkdirAll(forbiddenDir, 0755)

	allowedPaths := []string{allowedDir1, allowedDir2}

	t.Run("symlink to allowed location", func(t *testing.T) {
		targetFile := createTestFile(t, allowedDir1, "allowed.txt", "allowed content")
		linkPath := filepath.Join(tempDir, "safe_link.txt")
		createTestSymlink(t, targetFile, linkPath)

		err := ValidateSymlinkSecurity(linkPath, allowedPaths)
		if err != nil {
			t.Errorf("Expected no error for allowed symlink, got: %v", err)
		}
	})

	t.Run("symlink to forbidden location", func(t *testing.T) {
		targetFile := createTestFile(t, forbiddenDir, "forbidden.txt", "forbidden content")
		linkPath := filepath.Join(tempDir, "dangerous_link.txt")
		createTestSymlink(t, targetFile, linkPath)

		err := ValidateSymlinkSecurity(linkPath, allowedPaths)
		if err == nil {
			t.Error("Expected error for forbidden symlink")
		}
		if !strings.Contains(err.Error(), "not within any allowed base path") {
			t.Errorf("Expected 'not within allowed base path' error, got: %v", err)
		}
	})

	t.Run("regular file instead of symlink", func(t *testing.T) {
		regularFile := createTestFile(t, tempDir, "regular.txt", "content")

		err := ValidateSymlinkSecurity(regularFile, allowedPaths)
		if err == nil {
			t.Error("Expected error for regular file")
		}
		if !strings.Contains(err.Error(), "not a symbolic link") {
			t.Errorf("Expected 'not a symbolic link' error, got: %v", err)
		}
	})

	t.Run("broken symlink", func(t *testing.T) {
		brokenTarget := filepath.Join(tempDir, "nonexistent.txt")
		brokenLink := filepath.Join(tempDir, "broken.txt")
		createTestSymlink(t, brokenTarget, brokenLink)

		err := ValidateSymlinkSecurity(brokenLink, allowedPaths)
		if err == nil {
			t.Error("Expected error for broken symlink")
		}
		if !strings.Contains(err.Error(), "symlink resolution failed") {
			t.Errorf("Expected 'symlink resolution failed' error, got: %v", err)
		}
	})
}

// Test ValidateSymlinkSecurity with different scenarios
func TestValidateSymlinkSecurityComprehensive(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	// Create allowed directories for ValidateSymlinkSecurity
	allowedDir := filepath.Join(tempDir, "allowed")
	os.MkdirAll(allowedDir, 0755)

	// Create target file in allowed directory
	targetFile := createTestFile(t, allowedDir, "target.txt", "content")
	linkPath := filepath.Join(tempDir, "test_link.txt")
	createTestSymlink(t, targetFile, linkPath)

	// Test with ValidateSymlinkSecurity (requires allowlist)
	allowedPaths := []string{allowedDir}
	err1 := ValidateSymlinkSecurity(linkPath, allowedPaths)
	if err1 != nil {
		t.Errorf("ValidateSymlinkSecurity failed: %v", err1)
	}

	// Test regular file with ValidateSymlinkSecurity (should fail - not a symlink)
	err2 := ValidateSymlinkSecurity(targetFile, allowedPaths)
	if err2 == nil {
		t.Error("ValidateSymlinkSecurity should fail on regular file")
	}
	if !strings.Contains(err2.Error(), "not a symbolic link") {
		t.Errorf("Expected 'not a symbolic link' error, got: %v", err2)
	}

	// Test symlink to disallowed location
	forbiddenDir := filepath.Join(tempDir, "forbidden")
	os.MkdirAll(forbiddenDir, 0755)
	forbiddenFile := createTestFile(t, forbiddenDir, "forbidden.txt", "forbidden")
	badLinkPath := filepath.Join(tempDir, "bad_link.txt")
	createTestSymlink(t, forbiddenFile, badLinkPath)

	err3 := ValidateSymlinkSecurity(badLinkPath, allowedPaths)
	if err3 == nil {
		t.Error("ValidateSymlinkSecurity should fail for symlink to disallowed location")
	}
	if !strings.Contains(err3.Error(), "not within any allowed base path") {
		t.Errorf("Expected 'not within allowed base path' error, got: %v", err3)
	}
}

// Tests for RemoveSymlink

func TestRemoveSymlink(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	targetFile := createTestFile(t, tempDir, "remove_target.txt", "target content")

	t.Run("remove valid symlink", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "remove_link.txt")
		createTestSymlink(t, targetFile, linkPath)

		// Verify symlink exists
		if !fileExists(linkPath) {
			t.Fatal("Symlink was not created")
		}

		err := RemoveSymlink(linkPath)
		if err != nil {
			t.Fatalf("RemoveSymlink failed: %v", err)
		}

		// Verify symlink is removed
		if fileExists(linkPath) {
			t.Error("Symlink was not removed")
		}

		// Verify target still exists
		if !fileExists(targetFile) {
			t.Error("Target file was removed with symlink")
		}
	})

	t.Run("attempt to remove regular file", func(t *testing.T) {
		regularFile := createTestFile(t, tempDir, "regular_remove.txt", "content")

		err := RemoveSymlink(regularFile)
		if err == nil {
			t.Error("Expected error when trying to remove regular file")
		}
		if !strings.Contains(err.Error(), "not a symbolic link") {
			t.Errorf("Expected 'not a symbolic link' error, got: %v", err)
		}

		// Verify regular file is not removed
		if !fileExists(regularFile) {
			t.Error("Regular file was incorrectly removed")
		}
	})

	t.Run("non-existent symlink", func(t *testing.T) {
		nonExistentLink := filepath.Join(tempDir, "nonexistent_link.txt")

		err := RemoveSymlink(nonExistentLink)
		if err == nil {
			t.Error("Expected error for non-existent symlink")
		}
	})
}

// Tests for GetSymlinkTarget

func TestGetSymlinkTarget(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	targetFile := createTestFile(t, tempDir, "target_test.txt", "target content")

	t.Run("get relative symlink target", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "rel_target_link.txt")
		relativeTarget := "target_test.txt"
		createTestSymlink(t, relativeTarget, linkPath)

		target, err := GetSymlinkTarget(linkPath)
		if err != nil {
			t.Fatalf("GetSymlinkTarget failed: %v", err)
		}

		if target != relativeTarget {
			t.Errorf("Expected target %q, got %q", relativeTarget, target)
		}
	})

	t.Run("get absolute symlink target", func(t *testing.T) {
		linkPath := filepath.Join(tempDir, "abs_target_link.txt")
		absoluteTarget, _ := filepath.Abs(targetFile)
		createTestSymlink(t, absoluteTarget, linkPath)

		target, err := GetSymlinkTarget(linkPath)
		if err != nil {
			t.Fatalf("GetSymlinkTarget failed: %v", err)
		}

		if target != absoluteTarget {
			t.Errorf("Expected target %q, got %q", absoluteTarget, target)
		}
	})

	t.Run("regular file instead of symlink", func(t *testing.T) {
		regularFile := createTestFile(t, tempDir, "regular_target.txt", "content")

		_, err := GetSymlinkTarget(regularFile)
		if err == nil {
			t.Error("Expected error for regular file")
		}
		if !strings.Contains(err.Error(), "not a symbolic link") {
			t.Errorf("Expected 'not a symbolic link' error, got: %v", err)
		}
	})

	t.Run("non-existent symlink", func(t *testing.T) {
		nonExistentLink := filepath.Join(tempDir, "nonexistent_target_link.txt")

		_, err := GetSymlinkTarget(nonExistentLink)
		if err == nil {
			t.Error("Expected error for non-existent symlink")
		}
	})
}

// Integration tests

func TestSymlinkIntegration(t *testing.T) {
	tempDir := createTempDir(t)
	defer os.RemoveAll(tempDir)

	t.Run("complete symlink workflow", func(t *testing.T) {
		// Create target
		targetFile := createTestFile(t, tempDir, "workflow_target.txt", "workflow content")

		// Create relative symlink
		linkPath := filepath.Join(tempDir, "workflow_link.txt")
		err := CreateRelativeSymlink(targetFile, linkPath)
		if err != nil {
			t.Fatalf("CreateRelativeSymlink failed: %v", err)
		}

		// Verify it's a symlink
		isLink, err := IsSymlink(linkPath)
		if err != nil || !isLink {
			t.Fatalf("Created path is not a symlink: %v", err)
		}

		// Get symlink target
		target, err := GetSymlinkTarget(linkPath)
		if err != nil {
			t.Fatalf("GetSymlinkTarget failed: %v", err)
		}

		// Resolve symlink
		resolved, err := ResolveSymlink(linkPath)
		if err != nil {
			t.Fatalf("ResolveSymlink failed: %v", err)
		}

		// Validate security (allowing temp directory)
		allowedPaths := []string{tempDir}
		err = ValidateSymlinkSecurity(linkPath, allowedPaths)
		if err != nil {
			t.Fatalf("ValidateSymlinkSecurity failed: %v", err)
		}

		// Read content through symlink
		content := readFileContent(t, linkPath)
		if content != "workflow content" {
			t.Errorf("Content mismatch through symlink: %q", content)
		}

		// Remove symlink
		err = RemoveSymlink(linkPath)
		if err != nil {
			t.Fatalf("RemoveSymlink failed: %v", err)
		}

		// Verify symlink is gone but target remains
		if fileExists(linkPath) {
			t.Error("Symlink was not removed")
		}
		if !fileExists(targetFile) {
			t.Error("Target file was incorrectly removed")
		}

		t.Logf("Workflow completed: target=%s, link=%s, resolved=%s", target, linkPath, resolved)
	})

	t.Run("symlink portability test", func(t *testing.T) {
		// Create nested structure
		subDir := filepath.Join(tempDir, "subdir")
		os.MkdirAll(subDir, 0755)
		deepDir := filepath.Join(subDir, "deep")
		os.MkdirAll(deepDir, 0755)

		targetFile := createTestFile(t, tempDir, "portable_target.txt", "portable content")
		linkPath := filepath.Join(deepDir, "portable_link.txt")

		// Create relative symlink from deep directory to root
		err := CreateRelativeSymlink(targetFile, linkPath)
		if err != nil {
			t.Fatalf("CreateRelativeSymlink failed: %v", err)
		}

		// Verify target is relative
		target, err := GetSymlinkTarget(linkPath)
		if err != nil {
			t.Fatalf("GetSymlinkTarget failed: %v", err)
		}
		if filepath.IsAbs(target) {
			t.Error("Expected relative symlink for portability")
		}

		// Verify content is accessible
		content := readFileContent(t, linkPath)
		if content != "portable content" {
			t.Error("Content not accessible through deep relative symlink")
		}

		t.Logf("Portable symlink created: %s -> %s", linkPath, target)
	})
}

// Benchmark tests

func BenchmarkIsSymlink(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "fileops_bench_")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "bench.txt")
	if err := os.WriteFile(testFilePath, []byte("content"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	linkPath := filepath.Join(tempDir, "bench_link.txt")

	if isWindows() {
		b.Skip("Skipping symlink benchmark on Windows")
	}

	if err := os.Symlink(testFilePath, linkPath); err != nil {
		b.Fatalf("failed to create symlink: %v", err)
	}

	b.ResetTimer()
	for range b.N {
		IsSymlink(linkPath)
	}
}

func BenchmarkCreateRelativeSymlink(b *testing.B) {
	if isWindows() {
		b.Skip("Skipping symlink benchmark on Windows")
	}

	tempDir, err := os.MkdirTemp("", "fileops_bench_")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetFilePath := filepath.Join(tempDir, "bench_target.txt")
	if err := os.WriteFile(targetFilePath, []byte("content"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := range b.N {
		linkPath := filepath.Join(tempDir, "bench_link_"+string(rune(i))+".txt")
		CreateRelativeSymlink(targetFilePath, linkPath)
	}
}
