// Package repository provides abstractions for resolving central rule sources to local filesystem paths.
//
// This package implements the repository abstraction layer described in the GitHub-backed central repo
// specification. It handles both local directory sources and remote Git repository sources, providing
// a unified interface for accessing rule files regardless of their origin.
//
// # Architecture
//
// The package defines a Source interface that abstracts different types of rule repositories:
//   - LocalSource: Uses a local directory path directly
//   - GitSource: Clones/syncs from a remote Git repository to a local cache
//
// All sources resolve to a local filesystem path that can be used by the FileManager for
// consistent file operations, maintaining the existing public API contracts.
//
// # Path Resolution
//
// For remote Git repositories, this package provides simple, user-friendly path derivation:
//   - Clone path: <DefaultStorageDir>/<repo>
//   - Examples:
//   - git@github.com:user/rules.git → ~/.local/share/rulem/rules
//   - https://git.company.com/team/policies.git → ~/.local/share/rulem/policies
//
// This simple structure makes repositories directly visible and manageable by users outside the tool.
// Repository name conflicts are handled explicitly rather than through complex nested directory structures.
//
// # Security
//
// All path operations leverage the existing pkg/fileops security validators to prevent:
//   - Path traversal attacks
//   - Access to system/reserved directories
//   - Symlink security vulnerabilities
//
// Directory conflicts are resolved explicitly:
//   - If target directory exists, validate it's a git repository with the same remote URL
//   - If not the same repository, error and ask user to resolve manually
//   - This approach prioritizes safety and user control over automatic conflict resolution
//
// The package maintains the security-first approach established in the application architecture.
//
// # Usage Flow
//
//  1. Configuration specifies either local path or remote Git URL
//  2. Appropriate Source implementation is created
//  3. Source.Prepare() is called to resolve to local path
//  4. Local path is passed to FileManager for normal operations
//  5. All subsequent file operations work transparently
//
// # Design Goals
//
//   - Maintain existing FileManager public API unchanged
//   - Support offline operation with cached repositories
//   - Provide security through existing fileops validators
//   - Use simple, user-friendly repository paths for easy external management
//   - Handle directory conflicts explicitly and safely
//   - Support both local and remote repository sources seamlessly
//
// See the contracts/ directory for detailed interface specifications and expected behaviors.
package repository
