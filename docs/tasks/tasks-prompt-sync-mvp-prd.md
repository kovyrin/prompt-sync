# Prompt-Sync MVP – Task Tracker

_For full background and goals, see the [Product Requirements Document](prompt-sync-mvp-prd.md)._

## Relevant Files

- `internal/test/system/init_system_test.go` – End-to-end system tests for `prompt-sync init` (**write first**).
- `internal/cmd/root.go` – CLI entry-point (Cobra/Viper setup).
- `internal/cmd/init.go` – Implementation of `init` command.

- `internal/test/unit/config_loader_test.go` – Unit tests for configuration loader.
- `internal/config/loader.go` – Loads Promptsfile, local overrides, user config.

- `internal/test/unit/trusted_sources_test.go` – Unit tests for trusted source enforcement.
- `internal/security/trusted_sources.go` – Logic for allow-list and enforcement.

- `internal/test/contract/git_fetcher_contract_test.go` – Contract tests for Git fetcher interface.
- `internal/git/fetcher.go` – Git cloning & local cache management (go-git backend).
- `internal/git/fetcher_exec.go` – Exec backend using system git (auto-selected for large repos).

- `internal/test/contract/adapter_contract_test.go` – Contract tests for AgentAdapter interface.
- `internal/adapter/adapter.go` – Shared adapter interface definition.
- `internal/adapter/cursor/cursor_adapter.go` – Cursor adapter implementation.
- `internal/adapter/claude/claude_adapter.go` – Claude adapter implementation.

- `internal/test/integration/install_workflow_test.go` – Integration tests for `install` + `verify` end-to-end.
- `internal/workflow/install.go` – Install workflow orchestration.
- `internal/lock/lock_writer.go` – Lock file generation & parsing.
- `internal/gitignore/manager.go` – Managed `.gitignore` block injection & verification.

- `internal/test/unit/conflict_detector_test.go` – Unit tests for duplicate/​hash conflict detection.
- `internal/conflict/detector.go` – Conflict detection logic.

- `internal/test/unit/cli_commands_test.go` – Unit tests for `add`, `remove`, `update`, `list` commands.
- `internal/cmd/add.go`, `internal/cmd/remove.go`, `internal/cmd/update.go`, `internal/cmd/list.go` – Lifecycle commands.

- `internal/test/system/ci_mode_system_test.go` – System tests for CI/headless mode.
- `internal/ci/ci_guard.go` – CI safeguards (non-interactive flags, security levels).

### Notes

- **Outside-In TDD Reminder**: For every new feature, start with the highest-level failing test (system or contract). Write minimal code to pass before moving inward.
- **Safe Iteration**: One failing test at a time. Run `go test ./...` after each small change to keep feedback fast.
- **Test Layer Strategy**:
  • _System tests_ (`internal/test/system/`) execute the compiled binary via `os/exec` for real user flows.
  • _Contract tests_ (`internal/test/contract/`) define interfaces (e.g., GitFetcher, Adapter) before implementation.
  • _Integration tests_ (`internal/test/integration/`) exercise multiple layers together with fakes/mocks instead of real Git.
  • _Unit tests_ (`internal/test/unit/`) cover individual structs & methods.
- **Mocking & Stubs**: Use Go's `testing` + `httptest` & testify mocks. For Git, use a local bare repo fixture to avoid network calls.
- **CI Verification**: All verification tasks run `CI=true go test ./...` to ensure headless mode passes.
- **Managed `.gitignore` Block**: Implement idempotent insert/update logic; verify it every `install` and in tests.
- **Security Levels**: Security enforcement must block high-risk prompts in `--strict` or `CI=true`.
- **Code Standards**: Run `go vet ./...` and `golangci-lint run` (add to `Makefile`) before committing.
- **Watch Target (human-only)**: The Makefile includes `make watch` (uses `entr`) for humans; AI agents must NOT run it—run `make test` manually after each change instead.
- **Caching**: Git clones live under `$HOME/.prompt-sync/repos`; design Fetcher to support `--offline`.
- **Cache Directory Override**: Allow environment variable `PROMPT_SYNC_CACHE_DIR` or `--cache-dir` flag to override the default location.
- **Overlay Precedence**: Personal > project > org overlays must be applied when assembling rendered files; lower scopes are shadowed by higher ones.
- **Strict Mode**: `--strict` flag (or `CI=true`) escalates warnings (conflicts, hash drift, high security) to hard errors and aborts the operation.

## Tasks

- [x] 0. Development environment & CI scaffolding

  - [x] 0.1. Create `Makefile` with `test`, `lint`, and `watch` targets (`watch` re-runs `make test` on file changes using `entr`).

  - [x] 0.2. Add `.golangci.yml` configuration enabling recommended linters; ensure `make lint` runs `golangci-lint run`.

  - [x] 0.3. Add GitHub Actions workflow `.github/workflows/ci.yml` that runs `make test` and `make lint` on every push/PR.

  - [x] 0.4. Run `make test` and `make lint` locally to confirm green baseline.

- [ ] 1. Bootstrap CLI skeleton & `init` command

  - [ ] 1.1. Write failing system test `internal/test/system/init_system_test.go` asserting `prompt-sync init` scaffolds `Promptsfile` and managed `.gitignore` block.

  - [ ] 1.2. Run `go test ./internal/test/system/...` and confirm it fails.

  - [ ] 1.3. Create `internal/cmd/root.go` with Cobra root command and minimal `init` sub-command stub returning `not implemented`.

  - [ ] 1.4. Implement minimal logic in `internal/cmd/init.go` to create `Promptsfile` with template contents.

  - [ ] 1.5. Update system test to cover idempotency (should refuse to overwrite existing files unless `--force`).

  - [ ] 1.6. Run tests again; iterate until pass.

  - [ ] 1.7. Verify by running `go test ./...` and `prompt-sync init --help`.

- [ ] 2. Configuration loading & trusted source enforcement

  - [ ] 2.1. Write failing unit test `internal/test/unit/config_loader_test.go` for loading default sources from `Promptsfile` & user config precedence.

  - [ ] 2.2. Write contract test `internal/test/unit/trusted_sources_test.go` defining expected rejection of unknown Git URLs.

  - [ ] 2.3. Implement `internal/config/loader.go` with YAML parsing and precedence rules.

  - [ ] 2.4. Implement `internal/security/trusted_sources.go` enforcing allow-list with meaningful errors.

  - [ ] 2.5. Add edge-case tests (URL redirects, wildcards, `--allow-unknown`).

  - [ ] 2.6. Verify by running `go test ./internal/...`.

- [ ] 3. Prompt pack resolver: discovery, Git fetching, and local cache

  - [ ] 3.1. Write contract test `internal/test/contract/git_fetcher_contract_test.go` describing expected `GitFetcher` interface (Clone, Update, CachedPath).

  - [ ] 3.2. Stub `internal/git/fetcher.go` implementing the interface with no-op returns.

  - [ ] 3.3. Write unit tests for cloning into `$HOME/.prompt-sync/repos` using local fixture repos.

  - [ ] 3.4. Implement **go-git** backend in `internal/git/fetcher.go`; handle offline mode.

  - [ ] 3.5. Implement **exec-git** backend in `internal/git/fetcher_exec.go` that shells out to the system `git` binary with shallow/sparse clone support.

  - [ ] 3.6. Add backend factory & auto-selection logic (repo-size threshold or explicit `--git-backend`, `PROMPT_SYNC_GIT_BACKEND`); update contract tests to run against both backends.

  - [ ] 3.7. Add support for overridable cache directory via `PROMPT_SYNC_CACHE_DIR` env var and `--cache-dir` flag; update unit & contract tests.

  - [ ] 3.8. Verify by running `go test ./...` against both backends.

- [ ] 4. Adapter architecture & rendering engine (Cursor & Claude)

  - [ ] 4.1. Write contract test `internal/test/contract/adapter_contract_test.go` defining `Adapter` interface as per PRD.

  - [ ] 4.2. Stub `internal/adapter/adapter.go` with interface definition only.

  - [ ] 4.3. Write unit tests for Cursor adapter metadata merge algorithm (defaults → per-file → front-matter).

  - [ ] 4.4. Implement `internal/adapter/cursor/cursor_adapter.go` to render prompt files into `.cursor/rules/_active/`.

  - [ ] 4.5. Write unit tests for Claude adapter prefix resolution precedence.

  - [ ] 4.6. Implement `internal/adapter/claude/claude_adapter.go` to render prefixed markdown files.

  - [ ] 4.7. Verify by running `go test ./internal/adapter/...`.

- [ ] 5. Installation workflow: `install` & `verify` (lock file generation, .gitignore management, conflict detection)

  - [ ] 5.1. Write failing integration test `internal/test/integration/install_workflow_test.go` exercising `prompt-sync install` end-to-end with sample packs.

  - [ ] 5.2. Implement `internal/gitignore/manager.go` to insert/update managed block idempotently; add unit tests.

  - [ ] 5.3. Implement overlay precedence logic (personal > project > org) when staging packs for rendering; add unit & integration tests.

  - [ ] 5.4. Implement `internal/conflict/detector.go` scanning rendered outputs for duplicate basenames & hash drift; add tests.

  - [ ] 5.5. Implement `internal/lock/lock_writer.go` to write/update `Promptsfile.lock`; add tests for deterministic hashes.

  - [ ] 5.6. Implement strict-mode handling in install workflow (`--strict` flag or `CI=true`) to convert warnings into errors; update tests.

  - [ ] 5.7. Implement `internal/cmd/verify.go` exposing standalone `verify` command aliasing install-verify mode; add CLI tests.

  - [ ] 5.8. Implement `internal/workflow/install.go` orchestrating resolver → adapters → conflict scan → lock write; include `Verify` mode.

  - [ ] 5.9. Verify by running `go test ./...` and executing `prompt-sync install --strict` and `prompt-sync verify` on sample repo.

- [ ] 6. Package lifecycle commands: `add`, `remove`, `update`, `list`

  - [ ] 6.1. Write unit tests in `internal/test/unit/cli_commands_test.go` covering happy paths & error cases for each command.

  - [ ] 6.2. Implement `internal/cmd/add.go` to modify `Promptsfile` and trigger install.

  - [ ] 6.3. Implement `internal/cmd/remove.go` with safe removal & cleanup.

  - [ ] 6.4. Implement `internal/cmd/update.go` to bump unpinned branches and update lock file.

  - [ ] 6.5. Implement `internal/cmd/list.go` supporting `--outdated` & `--files` flags.

  - [ ] 6.6. Verify by running `go test ./...` and invoking each command against sample repo.

- [ ] 7. CI/headless mode safeguards & security enforcement

  - [ ] 7.1. Write failing system test `internal/test/system/ci_mode_system_test.go` ensuring `CI=true prompt-sync install` runs non-interactively and exits non-zero on conflict.

  - [ ] 7.2. Implement `internal/ci/ci_guard.go` to detect `CI=true` or `--yes` and force non-interactive mode.

  - [ ] 7.3. Extend security level checks to block high-risk prompts in strict mode; add unit tests.

  - [ ] 7.4. Hook CI guard into root command persistent pre-run.

  - [ ] 7.5. Verify by running `CI=true go test ./...` and executing `prompt-sync ci-install`.
