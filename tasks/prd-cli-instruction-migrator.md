# Product Requirements Document: CLI Instruction Migrator

## 1. Introduction/Overview

The CLI Instruction Migrator is a command-line tool designed to help developers easily move, copy, and synchronize instruction files (Markdown format) for various AI assistants (e.g., GitHub Copilot, Cursor, Claude, Gemini-CLI, Opencode). The tool provides a user-friendly terminal interface for managing instruction files, supporting both centralized and project-specific workflows, and ensuring compatibility with different IDEs and CLIs.

## 2. Goals

- Enable developers to browse, select, and manage instruction files from a centralized repository.
- Support copying or symlinking instruction files into project directories, with user choice.
- Allow editing of instruction files using the user’s preferred terminal editor.
- Provide configuration options for storage locations.
- Validate instruction files for correct format and prevent duplicates.
- Support multiple AI assistants and their required file formats/locations.

## 3. User Stories

1. As a developer, I want to browse available instruction files in my centralized repository, so I can select and manage them for my project.
2. As a developer, I want to copy or symlink instruction files into my project, so I can keep them in sync or customize them as needed.
3. As a developer, I want to edit instruction files using my preferred terminal editor, so I can quickly update rules without leaving the CLI tool.
4. As a developer, I want to configure or change the default storage location for instruction files, so I can organize my workspace as I prefer.
5. As a developer, I want the tool to validate instruction files and prevent invalid or duplicate entries, so I avoid errors in my AI assistant integrations.
6. As a developer, I want the tool to support multiple AI assistants and their required file formats/locations, so I can migrate rules between different tools and IDEs.

## 4. Functional Requirements

1. The tool must present a file viewer interface listing all available instruction files in the centralized repository, showing file names and descriptions.
2. The tool must allow users to select one or more instruction files to add/manage.
3. The tool must prompt the user to choose between copying or symlinking the selected file(s) into the current project/directory.
4. The tool must support the following AI assistants and their file formats/locations:
   - GitHub Copilot: `/.github/copilot-instructions.md`
   - Cursor: `/.cursor/rules` (this can be either in the root, or in the pwd)
   - Claude: `claude.md`
   - Gemini-CLI: `Gemini.md`
   - Opencode: `Agent.md` or `agents.md`
   - Windsurf: `.windsurfrules`
   - default location: current pwd where the tool is run, with the name being the same as the instruction file.
5. The tool must allow users to edit instruction files using their configured terminal editor (e.g., nano, vim).
6. On first startup, the tool must prompt the user to accept the default storage location or set a custom path, and allow changing this later.
7. The tool must validate that instruction files are in Markdown format and conform to the required structure (see [Copilot instructions file structure](https://code.visualstudio.com/docs/copilot/copilot-customization#_instructions-file-structure)).
8. The tool must prevent overwriting existing files without explicit user confirmation.
9. The tool must display clear error messages for invalid files or operations.
10. The tool must use Go charm libraries (e.g., bubbletea, lipgloss) for the terminal UI.

## 5. Non-Goals (Out of Scope)

- The tool will not provide backup or versioning of instruction files.
- The tool will not support batch operations or automation via command-line arguments (for now).
- The tool will not support non-Markdown instruction file formats.
- The tool will not handle team/shared usage scenarios.

## 6. Design Considerations

- Use Go charm libraries (bubbletea, lipgloss, etc.) for a modern, interactive terminal UI.
- The file viewer should display both file names and descriptions (parsed from the Markdown frontmatter or content).
- The tool should be extensible to support additional AI assistants and formats in the future.

## 7. Technical Considerations

- The tool should be implemented in Go for cross-platform compatibility.
- Use the user’s `$EDITOR` environment variable to determine the default editor.
- Store configuration (e.g., storage location) in a user-accessible config file (e.g., YAML or JSON).
- Validate Markdown files using a schema or regular expressions based on the Copilot instructions file structure.

## 8. Success Metrics

- Users can successfully browse, select, and migrate instruction files between projects and AI assistants.
- Instruction files are correctly recognized and used by the target AI assistant after migration.
- No invalid or duplicate instruction files are created.
- User feedback indicates the tool is easy to use and reduces manual file management.

## 9. Open Questions

- What are the exact file locations/formats for Gemini-CLI and Opencode?
- Should the tool support additional AI assistants in the future?
- Should there be an option to export/import configuration settings?
- How should the tool handle updates to instruction file formats or requirements from supported AI assistants?
