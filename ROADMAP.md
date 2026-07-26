# rulem — Project State & Roadmap

_Last reviewed: July 2026, after the production-readiness PR batch (#4–#16)._

## Where the project stands

- **multiple-repos is fully merged.** `main` contains all 11 commits of the
  feature branch (it was rebased, so the branch shows different hashes but an
  identical tree). The local `multiple-repos` branch can be deleted.
- **Build and tests are green** on Go 1.26 (`go build ./...`, `go test ./...`
  across all 15 packages).
- **Core flows work end-to-end** (verified in a live terminal): setup config →
  main menu → Save rules (CWD scan, split filepicker + preview) → Import rules
  (multi-repo scan with repo badges, copy/symlink, editor formats) → settings
  menu → `rulem mcp` server.
- The **filepicker/preview overflow bug is fixed** (picker was created after
  startup with hardcoded 100×30 dims and each pane sized to the full window;
  it now sizes itself on creation and on resize, and re-wraps previews).
  See `docs-rendering-and-layout.md` for the full write-up.
- **A production-readiness PR batch is open/merged** covering CI, security
  fixes, self-healing repos, a repo-status screen, subdirectory saves, editor
  target refresh, and dependency upgrades (details below).

## Immediate cleanup (before calling it production-ready)

- [ ] **Move `credential` and `creds` out of the repo root and revoke them.**
  They are exported Snapcraft store macaroons (Aug 2025, default 1-year expiry
  — possibly still valid). Never committed, and now gitignored defensively,
  but secrets don't belong in a working tree. Re-export per `RELEASING.md`
  when needed.
- [x] Delete stale build artifacts (`*.test` binaries, old `dist/`) — PR #6.
- [ ] Decide a home for stray docs: `Big test.md` (177 KB fixture?),
  `bubbles_cheatsheet.md`, `devCentral/`.
  (`settings-menu-flows.md` is resolved: replaced by a corrected
  `internal/tui/settingsmenu/README.md` — PR #13.)
- [x] **Add a CI workflow** — PR #4 (`ci.yml`: vet/build/test -race + golangci-lint
  with only-new-issues gating; pre-existing lint issues left for later — 142 at
  the time, 90 today across 11 packages, 28 of them in `internal/tui/settingsmenu`).
- [x] Fix config-path docs (README/`config.go` said `~/.config/rulem`; actual
  macOS path is `~/Library/Application Support/rulem/config.yaml`) — PR #5.
- [ ] Renew release tokens when cutting the next release (see `RELEASING.md`):
  `HOMEBREW_TAP_GITHUB_TOKEN`, `SNAPCRAFT_STORE_CREDENTIALS`.
- [ ] Bump Go toolchain to 1.26.1+ to clear the remaining stdlib `govulncheck`
  advisories (all dependency-level findings, incl. GO-2026-5496 in go-git,
  were fixed in PR #16).

## Known issues & robustness work

- [x] **Stale repository paths are now self-healing** — PR #10: failed repos
  stay listed as "⚠️ unavailable" for repair in settings instead of erroring
  the whole app; healthy repos keep working; GitHub repos re-clone themselves.
- [x] "Fetch rules from Github" placeholder replaced by a **repository status
  screen** with manual refresh-all — PR #11 (stacked on #10).
- [x] Central repo **subdirectories**: scanning was already recursive (tested);
  saving INTO subdirectories now supported (`backend/api-rules.md`) — PR #14.
- [x] **Repository sync actually updates the working tree now** — PR #9 fixed
  fetch never resetting the checkout (users never received upstream changes),
  implemented real shallow clones (`Depth: 1`), and pinned dirty-repo-skip
  behavior with tests.
- [x] **pkg/fileops security fixes** — PRs #20–#25 (**not** #15, which was an
  earlier independent attempt at the same three defects and was never merged;
  close it): AtomicCopy predictable temp-path symlink attack (S1), dead symlink
  guard in ValidateFileInDirectory (S2), dirscan symlink misclassification
  (S3), plus the `..` matcher (S4), the markdown HTML-injection scan (S5), the
  fixed probe name (S6) and ~455 lines of dead API. Full review in
  `reports/fileops-review.md`.
- [ ] **pkg/fileops → `os.Root` migration** — four stacked PRs per
  `reports/fileops-review.md` §9: introduce a `fileops.Dir` handle, swap its
  internals to `os.Root`, migrate the call sites, then delete the nine
  string-based validators. Target: ~770 → ~250 non-test lines, exported
  surface 23 → ~10.
- [ ] `tui.go` menu sizing uses a magic `v := 14 // footer margins`; derive it
  from rendered header/footer heights like the filepicker now does.
- [x] Windows: dropped — goreleaser windows bits removed (PR #6).
- [x] Filepicker polish — PR #8 (basename in messages, humanized sizes,
  word-wrapped plain previews) and PR #7 (repo-name row in multi-repo mode).
- [ ] **Filepicker known bugs from deep review** (`reports/filepicker-review-and-upstreaming.md`):
  keybindings fire while typing in the filter (`f`/`g`/`q`/`enter` hijacked);
  parents intercept `q`/`esc` and break filtering; data race in the render
  command closure; stale-width preview after resize during an in-flight
  render; negative pane widths below ~41 cols. Fix these before any
  publishing/upstreaming of the component.
- [ ] **Settings menu bugs from README validation** (PR #13 "Known issues"):
  failed manual refresh silently swallowed (`refreshCompleteMsg` clears the
  error unconditionally); delete-repo never dirty-checks; `pluralize` renders
  "repositoryies"; assorted dead code.

## Feature roadmap

### Near term (rounds out the current story)

1. **GitHub/gist fetch** (todo.md 7.1) — download a single rule file from a
   repo/gist URL into the central repo; reuse the existing PAT + go-git
   plumbing from the multi-repo work. (Note: full-repo fetching is done —
   that's the multi-repo feature; this item is only the one-off URL/gist grab.)
2. **Headless CLI commands** (todo.md 8.0) — `rulem save <file>`,
   `rulem import <rule> --editor cursor --link`, `rulem list` for scripts and
   CI; the TUI stays the interactive front end. Cobra is already wired.
3. ~~More editor targets~~ **Done for the current set** — PR #12: AGENTS.md is
   now the first/recommended default (open standard, 25+ tools), Cursor fixed
   to emit `.mdc`, all docs URLs verified. Remaining breadth: Windsurf, Zed,
   Kiro entries when demand shows up — each is a ~10-line registry entry + test.
4. ~~Repo health check & repair~~ **Done** — PR #10 self-healing + PR #11
   status screen.

### Mid term (differentiators)

5. **MCP write-back tools**
   *What:* Today `rulem mcp` only serves rules as read-only tools. This adds
   `save_rule` / `update_rule` MCP tools so an assistant can persist new or
   refined instructions into the central repo from inside a session — turning
   rulem into a two-way memory for AI coding agents, which is the strongest
   differentiator in this list.
   *New UI:* None in the TUI initially; the surface is MCP tools. Later, a
   "recently written by agents" filter in the picker.
   *Implementation plan:*
   1. Define tool schemas (`save_rule{name, content, repo?}`,
      `update_rule{name, content}`) in `internal/mcp`.
   2. Route writes through the existing `FileManager.CopyFileToStorage` path
      (content arrives as a string → write to temp file → atomic copy), incl.
      the new `SanitizeRelativePath` subdirectory support.
   3. Restrict writes to one configured "writable" repo (config flag on
      `RepositoryEntry`); refuse GitHub repos with dirty state.
   4. Tests against the MCP server harness; document in README + a security
      note (an LLM can now write files — scope it).
   *Effort:* ~1-2 days.

6. **Rule search**
   *What:* Full-text/fuzzy search across all repos' rule contents from the
   main menu — the filepicker filter only matches file names today. Finds
   "which rule mentions ruff?" instantly across N repos.
   *New UI:* A "Search rules" main-menu item opening a textinput + results
   list (same split-pane preview as the filepicker, reusing that component).
   *Implementation plan:*
   1. Index step in `filemanager`: on scan, read file contents (bounded size)
      into memory alongside `FileItem`.
   2. Match with a simple case-insensitive substring/fuzzy ranker
      (bubbles' fuzzy filter or `sahilm/fuzzy`) over name + content.
   3. New `searchmenu` TUI model embedding the existing filepicker list +
      preview panes; highlight matched lines in the preview.
   4. Tests for the ranker; live QA on a large repo for latency.
   *Effort:* ~2-3 days (mostly UI polish).

7. **Frontmatter schema + validation**
   *What:* Define a YAML frontmatter schema for rule files (name, description,
   targets, tags) and validate on save/import, surfacing problems in the
   picker. This is the prerequisite for good MCP tool generation (tool
   descriptions come from `description`) and for future conflict detection.
   *New UI:* Validation warnings inline in the filepicker description row and
   a "⚠ invalid frontmatter" badge; save flow gains an optional prompt to add
   missing fields.
   *Implementation plan:*
   1. Define the schema + parser in a new `internal/rulespec` package
      (goldmark-frontmatter or `adrg/frontmatter`); everything optional so
      bare files stay valid.
   2. Parse during scans; attach `Spec`/`SpecError` to `FileItem`.
   3. Surface in picker rows + preview header; block nothing (warn-only).
   4. Use `description` in `internal/mcp` tool registration (replaces
      filename-derived descriptions).
   5. Table-driven parser tests + fixture files.
   *Effort:* ~2 days.

8. **Sync UX for GitHub repos**
   *What:* Close the write-back loop for GitHub-backed repos: auto-commit (and
   optionally push) on save, dirty-state surfacing in the main menu, and
   conflict messaging. Pull-before-import and the manual refresh screen
   (PR #11) already exist; this adds the outbound half.
   *New UI:* Dirty badge on the main menu ("2 unpushed changes"), a
   commit+push confirm step at the end of the save flow, and clearer conflict
   errors in the status screen.
   *Implementation plan:*
   1. Add `CommitAndPush(path, message)` to `internal/repository`
      (go-git worktree add/commit/push with stored PAT).
   2. Save flow: if target repo is GitHub-backed, offer "[c]ommit & push /
      [l]ocal only" after a successful save.
   3. Extend the status screen rows with ahead/behind counts
      (`remote-tracking vs HEAD`).
   4. Conflict path: push rejection → surface "diverged; refresh first".
   5. Tests against local bare-repo fixtures (the harness from
      `git_sync_test.go` already does exactly this).
   *Effort:* ~3-4 days (push auth + conflict paths are the long tail).

9. **Rule diff/versions**
   *What:* Show a rule's git history and diff the central copy against the
   file actually sitting in an editor's directory (detect drift for copied —
   not linked — imports). Answers "did I tweak this rule locally after
   importing it?"
   *New UI:* A "history" action in the picker (list of commits touching the
   file) and a side-by-side/unified diff view (new viewport screen); an
   "out of sync" badge in the import flow when the editor copy differs.
   *Implementation plan:*
   1. `internal/repository`: `FileLog(path)` via go-git commit iteration
      filtered by path; `FileAtCommit(path, hash)`.
   2. Diff rendering with a small pure-Go diff lib
      (`sergi/go-diff` or `hexops/gotextdiff`) into a styled viewport.
   3. Drift detection: on import listing, hash-compare central copy vs
      editor-target file when the target exists.
   4. New `historymenu` model wired from the picker's action row.
   5. Tests for log/diff helpers on fixture repos.
   *Effort:* ~3-4 days.

### Long term / ideas parked in todo.md

- Conflict detection between rules targeting the same editor.
- A "manage MCP" menu to choose which files are exposed as tools.
- Templates/starter packs of common rule sets.
- Demo GIFs with `vhs` for the README — **in progress** (tapes + demo
  sandbox authored; rendering needs `ttyd` + `ffmpeg` installed).
- Docs site if adoption grows.

## Editors / AGENTS.md findings (July 2026 research)

- **AGENTS.md is now the default and recommended target** (PR #12). It is a
  Linux-Foundation-stewarded open standard read by 25+ tools (Cursor, Copilot
  coding agent, Gemini CLI, Zed, Jules, …). The import UI pre-selects it.
- **Claude Code does not read AGENTS.md** — its native file is `CLAUDE.md`
  (kept as a separate entry). Bridge trick if ever needed: a `CLAUDE.md`
  containing `@AGENTS.md`.
- **Cursor ignores plain `.md` under `.cursor/rules/`** — only `.mdc` is read.
  The Cursor entry now saves `foo.md` → `.cursor/rules/foo.mdc`. Verbatim
  copies carry no frontmatter, so they act as manual/@-referenced rules;
  always-on rules should go to AGENTS.md (Cursor reads it natively).
- **Copilot**: repo-wide `.github/copilot-instructions.md` works verbatim;
  path-scoped `.instructions.md` files need `applyTo` frontmatter rulem can't
  inject — the UI now says so.
- Several vendor docs URLs had rotted (VS Code docs reorg, `docs.cursor.com` →
  `cursor.com/docs`, `docs.anthropic.com` → `code.claude.com`); all refreshed
  and verified in PR #12.

## Publishing / extraction verdicts (July 2026 reviews)

Full reports live in `reports/`:

- `reports/fileops-review.md` — pkg/fileops security review (3 confirmed
  defects, fixed in PRs #20–#22), publish verdict (not as-is; worthwhile as a
  CV item after an `os.Root` rebuild), a concise Go-package publishing guide,
  and §9: the four-PR `os.Root` migration plan.
- `reports/filepicker-review-and-upstreaming.md` — component review (3 HIGH
  bugs to fix first) + upstreaming path: charmbracelet/bubbles does not accept
  new core components; the sanctioned route is a standalone repo listed on
  `charm-and-friends/additional-bubbles` (needs README + VHS GIF).
- `reports/git-scaffolding-extraction.md` — don't extract now; the only
  worthwhile unit is a `gitmirror` package (~24-32h of work); the higher-
  leverage move is a blog-post write-up of the fetch-vs-worktree bug.

## Release checklist pointer

Publishing to Homebrew + Snap is documented in [RELEASING.md](./RELEASING.md)
(tokens, multipass snapcraft login, tagging, verification).
