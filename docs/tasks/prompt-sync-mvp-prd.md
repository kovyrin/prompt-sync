# Product Requirements Document: Prompt-Sync MVP

_Last updated: 2025-06-24_

---

## 1. Overview

Prompt-Sync is a **package-manager-style CLI tool** that installs, version-locks, and audits AI prompts & rules across multiple developer agents (e.g. Cursor, Claude, Copilot) **without relying on public registries**. Think of it as **"npm for trusted AI prompts"**: prompt packs live in normal Git repositories, are installed locally into predictable locations, and are tracked in a lock file for deterministic builds.

The tool solves three persistent problems:

1. **Prompt Drift & Duplication** – Copy-pasting prompt files between projects leads to stale versions, naming collisions, and divergent CI behaviour.
2. **Security & Compliance** – Enterprises cannot adopt community-hosted prompt managers that pull unvetted content over the Internet; they need zero outbound traffic, deterministic hashes, and auditability.
3. **Multi-Agent Friction** – The same logical prompt often needs to exist in several different on-disk formats (Cursor rule, Claude slash-command, Copilot instruction). Manual duplication is error-prone and hard to keep in sync.

---

## 2. Goals & Non-Goals

**Goals**

- Delightful developer experience: fast, friendly CLI with Rust-compiler-style error messages and guard-rails to prevent mistakes.
- Deterministic installs reproducible in local dev **and** CI/CD.
- Private/org → project → personal overlay model with clear precedence.
- Support at least **Cursor** and **Claude** from a single prompt pack; easy path to add Copilot, Codeium, etc.
- Zero outbound network requests by default; relies on existing Git remotes.
- Trust-based security model preventing prompt-injection attacks.

**Non-Goals**

- Public "npm-style" registry on day 1 (can arrive later).
- Fine-grained dependency solver (simple semver/tags are enough).
- GUI/Web dashboard (CLI only in v0.x).
- Automatic prompt generation or analytics.
- Windows support before v1.0 (focus macOS/Linux).

---

## 3. Key Concepts

- **Prompt Pack** – A Git-versioned directory of markdown prompts/rules plus YAML front-matter metadata.
- **Adapter** – Go plug-in that renders a prompt pack into the on-disk format expected by an AI agent (e.g. Cursor rule files).
- **Scope / Trust Level** – `org → project → personal` (higher scopes shadow lower ones).
- **Promptsfile** – YAML configuration committed to the repo that declares trusted sources and selected packs.
- **Promptsfile.lock** – Auto-generated file mapping _pack → adapter → file hash_ for deterministic builds.
- **Install Path** – `.ai/prompts/` inside the project by default; adapter decides agent-specific target dirs.

---

## 4. User Personas & Stories

### 4.1 Core Workflows

1. **New Project Setup** – "I clone a project, run `prompt-sync install`, and have all company-standard AI rules within 30 seconds."
2. **Personal Customisation** – "I can layer my own Git workflow prompts on top of corporate rules without affecting teammates."
3. **Prompt Discovery** – "I browse approved packs and add `shopify/graphql@stable` to my project."
4. **Version Control** – "`prompt-sync update` bumps tracked branches and rewrites the lock file so everyone stays in sync."
5. **Multi-Agent Consistency** – "The same `git.commit` rule appears in Cursor, Claude, and Copilot formats automatically."

### 4.2 Administrative Workflows

6. **Prompt Publishing** – "I publish updates to our org repo; colleagues receive them via `prompt-sync install`."
7. **Audit & Compliance** – "Security scans the lock file diff in PRs to verify no unreviewed prompts enter production."
8. **CI / Remote Agent Provisioning** – "In a GitHub Actions workflow, `prompt-sync ci-install` fetches the exact rules for the repository, so the AI agent running inside the job uses the same prompts the team sees locally."

---

## 5. Functional Requirements

### 5.1 Configuration Files

1. **Promptsfile** (committed) – declares trusted sources and rulesets.
2. **Promptsfile.lock** (committed) – exact resolved commits & file hashes.
3. **Promptsfile.local** (git-ignored) – per-developer overrides.
4. `~/.prompt-sync/config.yaml` – user-level default sources and packs.

Full schema is preserved from the original draft (see Appendix A).

### 5.1.a Headless / CI Execution

- All commands must run non-interactively when `CI=true` or when `--yes` is supplied.
- `prompt-sync ci-install` (alias for `install --yes --strict`) is provided for clarity in CI scripts - no need to add extra flags.
- Exit with non-zero status if any operation would require user input.

### 5.1.b Claude Command Prefix Resolution

Prompt-Sync uses a filename prefix to distinguish generated **Claude** slash-command files and avoid clashes between different sources. The prefix for each command is determined at install-time by the following precedence:

1. `sources[i].claude_prefix` – explicit override on a source entry.
2. `config.claude_prefix` – optional repo-wide fallback.
3. `sources[i].name` – automatic default (kebab-cased).

This "convention over configuration" means that in the common case no additional settings are required: a source named `shopify` will automatically produce `shopify-*` command files, while `personal` yields `personal-*`. Teams can still override the prefix when they need a shorter or legacy tag without affecting other sources.

### 5.2 Core Commands (MVP)

- `prompt-sync init` – Scaffold `Promptsfile` and `.gitignore` entries.
- `prompt-sync install [--agents=cursor,claude] [--strict]` – Resolve packs, render via adapters, update lock file.
- `prompt-sync add <source/pack[@ref]>` – Add a pack and update the lock file.
- `prompt-sync update [<pack>]` – Pull latest commits on tracked branches.
- `prompt-sync remove <pack>` – Remove a pack and clean rendered files.
- `prompt-sync list [--outdated] [--files]` – Show installed packs, versions, and optionally files.
- `prompt-sync verify` – CI-oriented command that re-renders and fails on drift.

### 5.3 Adapter API (internal)

```go
// Simplified interface
interface AgentAdapter {
    Name() string              // "cursor", "claude", "copilot", ...
    Detect() bool              // auto-enable if agent present
    TargetDir(scope Scope) string
    Render(pack PromptPack, scope Scope) ([]RenderedFile, error)
    Verify(files []RenderedFile, mode Strictness) error
}
```

Bundled adapters in v0.1: Cursor, Claude (slash-command & markdown).

### 5.3.a Front-Matter Precedence & Metadata Strategy

To support multiple agents (Cursor, Claude, Copilot …) while keeping prompt files clean, the Cursor adapter merges metadata from three layers - closest to the file wins:

1. **Ruleset defaults** – top-level `defaults:` block inside `metadata.yaml`
2. **Per-file overrides** – entries under `files:` in the same `metadata.yaml`
3. **Inline front-matter** – YAML header embedded directly in the prompt file

Merging algorithm (Cursor adapter):

```
effective := metadata.defaults
if metadata.files[filename] exists:
    effective = merge(effective, metadata.files[filename])
if file has front-matter:
    effective = merge(effective, file.frontMatter)
render prompt with effective front-matter
```

`merge` performs a shallow overwrite; later layers replace scalar values or array entries. Setting a value to `null` in an override removes it (opt-out mechanism).

Example `metadata.yaml`:

```yaml
defaults:
  alwaysApply: true
  globs: ["**/*.rb"]

files:
  coding-style.mdc:
    description: "Ruby style guide"
    globs: ["**/*.rb", "**/*.erb"]
    alwaysApply: false # overrides default
  git-workflow.md:
    globs: ["**/*.md"] # inherits alwaysApply=true
```

This strategy keeps raw prompts portable, allows bulk edits via the manifest, and remains backward-compatible with existing Cursor-specific front-matter.

### 5.4 Conflict Detection

Conflict handling rules (Dev Mode → `--strict`/CI):

- **Duplicate output filename after render** – Warn & list shadows → Abort install.
- **Hash drift vs lock** – Warn → Abort.
- **Version-solver clash** – Abort → Abort.

Personal > project > org precedence. Only _winning_ files enter the lock; personal files never enter the shared lock.

### 5.5 Performance Targets

- Install ≤ 5 seconds for 100 prompt files (post-clone).
- CLI start-up ≤ 200 ms on cold start.

### 5.6 Distribution & Release Automation

Prompt-Sync binaries must be easy to **install**, **update**, and **publish** without requiring developers to remember multi-step commands.

**Requirements**

1. **One-liner install / update**

   - Primary path: `go install github.com/kovyrin/prompt-sync/cmd/prompt-sync@latest`.
   - Convenience wrapper: `make install` (delegates to the same `go install` line).
   - Works on any machine with Go ≥1.22 and internet access to GitHub.

2. **Tagged releases + automated artifacts**

   - `make release VERSION=vX.Y.Z` – verifies clean git tree, tags the commit, pushes the tag.
   - A GitHub Actions workflow (`.github/workflows/release.yml`) watches `v*.*.*` tags and runs **GoReleaser**.
   - GoReleaser cross-compiles macOS & Linux binaries for `amd64` and `arm64`, packages them as `tar.gz`, and uploads checksums.

3. **Snapshot builds for local testing**

   - `make snapshot` runs `goreleaser release --snapshot --clean` producing local artifacts suffixed with `+dirty` (not published).

4. **Optional Homebrew tap (post-MVP)**
   - Tap scaffolding is commented in `.goreleaser.yml`; enable once `homebrew-tap` repo is ready.

**Non-Goals**

- Windows binaries (deferred until v1.0).
- Automated push on every commit to `main` – only explicit semver tags trigger releases.

**Implications**

- Removes the open question around "Binary Distribution" by adopting the GoReleaser + GitHub Releases approach.
- Keeps the MVP toolchain Go-native; no additional package registry required.

---

## 6. System Architecture

### 6.1 Pack Layout & Metadata

```
my-pack/
├── prompts/
│   └── git.commit.md   # contains front-matter
└── README.md
```

```yaml
---
title: "Git Commit Message"
kind: rule
targets: [cursor, claude.cmd]
scope: git
version: 1.0.0
security: low
---
<template body>
```

### 6.2 Install Flow

1. Resolve packs → local Git clones (or internal registry in future).
2. Stage personal/project/org overlays (respect precedence).
3. Invoke each active adapter's `Render` to materialise files.
4. Run conflict scanner across rendered outputs.
5. Write or update `Promptsfile.lock` with file hashes.
6. `prompt-sync verify` re-scans workspace for drift in CI.

### 6.3 Directory Structure (after install)

```
project-root/
├── Promptsfile
├── Promptsfile.lock
├── .ai/prompts/            # raw packs (git-ignored)
│   ├── shopify/
│   └── personal/
│   ├── .cursor/
│   │   └── rules/_active/      # rendered by Cursor adapter (git-ignored build artefacts)
│   └── .claude/
│       └── commands/           # rendered by Claude adapter, files prefixed and git-ignored
```

### 6.3.a Git-ignore Rules Managed by Prompt-Sync

During `prompt-sync init` (and kept up-to-date on every `install`), the tool ensures the repository's `.gitignore` contains a **managed block** so that build artefacts never end up in commits while any hand-written prompts remain version-controlled.

Default entries (prefix `ps` shown; honouring per-source or global Claude prefix resolution):

```
# >>> prompt-sync managed block – DO NOT EDIT BY HAND
.ai/prompts/
.cursor/rules/_active/
.claude/commands/ps-*
# <<< prompt-sync managed block
```

Key points:

• Only generated Claude commands (`<prefix>-*.md`) are ignored - user-authored files without the prefix still appear in git status.
• The managed block is idempotent: prompt-sync rewrites it safely without touching the rest of `.gitignore`.
• Teams can change the prefix via `sources[i].claude_prefix` (preferred) or fallback `config.claude_prefix`; the ignore line will update automatically.
• On every `install` or `update`, prompt-sync verifies that the **entire managed block remains present and unmodified**. If it cannot inject or repair the block it aborts with a clear error - no adapter is allowed to generate files that could end up committed.

### 6.4 Storage & Caching

- Git repos cloned once into `~/.prompt-sync/repos/` (acts as cache).
- Offline installs from cached repos are opt-in (use `--offline` flag); network fetch remains the default.

---

## 7. Security & Compliance

- **Trusted Sources Only** – prompt-sync refuses unknown Git remotes unless explicitly allowed.
- **Zero Credential Storage** – delegates auth to user-configured Git (SSH keys, tokens).

### 7.1 Prompt Security Levels

Each prompt may declare a `security` field in its front-matter (or in `metadata.yaml`). This optional label helps organisations gate risky prompts in CI and during updates.

- **low** – Read-only or advisory content (style guides, naming conventions). No code execution, no tool calls.
- **medium** – Generates or modifies source code or configs; could introduce bugs if mis-used (e.g., "generate GraphQL schema", "refactor component").
- **high** – Executes commands, manipulates git history, deletes files, or otherwise has side-effects that could exfiltrate data or corrupt the project.

Enforcement in MVP:

1. **Install / Update** – Warn when a prompt's security level exceeds the user-defined threshold; block in `--strict` mode.
2. **Lock Verification** – Fail CI if a prompt's security level changes without explicit approval.

### 7.2 Trusted Source Enforcement

Prompt-Sync treats each Git repository that hosts prompt packs as a _source_ and refuses to interact with repositories that have not been explicitly declared.

Why this matters:

- **Prompts can execute code** – In modern IDE agents, a malicious prompt is equivalent to running untrusted scripts.
- **Repo spoofing is easy** – Attackers can clone/fork well-known prompt packs under look-alike names and trick users into adding them.
- **Scale** – Large organisations cannot manually review every incoming prompt at merge-time; an allow-list keeps the surface area small.

Mechanism:

1. **Allow-list in configuration** – Only Git URLs listed in `sources:` blocks across _Promptsfile_, _Promptsfile.local_, or `~/.prompt-sync/config.yaml` are eligible.
2. **URL pinning** – Prompt-Sync resolves the canonical clone URL (after redirects); if it differs from the allow-listed value, installation aborts.
3. **Interactive override** – A developer may bypass once with `--allow-unknown`, but the CLI prints a prominent warning and records the event in the audit log. This flag is disabled when `CI=true` or `--strict`.
4. **Policy integration** – Organisations can pin allowed domains (e.g., `github.com:shopify/*`) in a future policy file and block any source outside that namespace.

Example `Promptsfile` snippet:

```yaml
sources:
  - name: shopify
    repo: git@github.com:shopify/ai-prompts.git
    claude_prefix: shp # Optional: override default "shopify" prefix
  - name: personal
    repo: git@github.com:kovyrin/my-prompts.git # allowed
# Any other repo will be rejected unless --allow-unknown is supplied.
```

This mandatory allow-list provides a baseline guarantee against supply-chain attacks while remaining flexible for personal experimentation.

---

## 8. Version Management

- Default: track remote's default branch.
- References: `pack`, `pack@v1.2.0`, `pack@stable`, `pack@<commit>`.
- Lock file always records exact commit SHA **and** rendered file hashes.
- `prompt-sync update` only moves unpinned branches.

---

## 9. Roadmap

MVP scope (**Phase 0.1**):

• Core package management: `init · install · add · verify`
• Cursor & Claude adapters with prefix-based file naming
• Lock file generation & duplicate-basename scanner
• CLI flags for headless/CI execution
• Trusted-source enforcement & basic security levels

---

## 10. Open Questions

1. **Binary Distribution** – goreleaser + Homebrew vs internal Artifactory?
2. **Registry Format** – simple static JSON vs OCI-style artifact registry?

---

## 11. Future Ideas & Enhancements Backlog

This section captures aspirational features **beyond the scoped roadmap** so they are not lost during focused MVP work. Items here have no committed timeline; they serve as a brainstorming backlog.

**Developer Experience (Top Priority)**

- Advanced conflict-resolution UI (three-way merge)
- Interactive resolver wizard for per-file shadow decisions
- Export "materialise" command to generate rendered artefacts for legacy tooling
- AI-powered prompt quality linter (local or proxied OpenAI, see Security notes)
- AI-assisted smart conflict resolution hints

**Security & Compliance**

- Prompt-fuzz security scanner (detect jailbreak/destructive prompts)

**Registry & Distribution**

- Offline registry mirror for air-gapped environments

**Observability & Insights**

- Semantic duplication detector (find overlapping prompts)

Contributors are encouraged to move items from this backlog into the Roadmap once scope, effort, and priority are clearly defined.

> Note on AI integrations: LLM helpers are **off by default**. When enabled, they can run (a) locally, (b) through a company-controlled proxy, or (c) directly against a public endpoint such as api.openai.com **if explicitly opted-in**. Prompt-Sync exposes `--llm-endpoint` / `llm_endpoint:` and `enable_llm: true` flags so teams can route traffic via their approved gateway or point to another URL at their own risk.

---

## 12. Conclusion

Prompt-Sync brings proven package-management principles to AI prompt engineering while meeting the stringent security and compliance requirements of modern enterprises. By treating prompts as dependencies - installed locally, version-locked, and audited - we enable scalable, maintainable, and shareable AI assistance across organisations **without exposing developers to untrusted content**.

The MVP delivers immediate value (deterministic installs & multi-agent rendering) and lays a solid foundation for advanced enterprise-grade features in subsequent releases.

---

## Appendix A – Promptsfile Example

```yaml
version: 1

sources:
  # IMPORTANT: Only add trusted sources! Untrusted prompts can lead to
  # prompt injection attacks with serious security consequences.

  - name: shopify
    repo: git@github.com:shopify/ai-prompts.git
    claude_prefix: shp # Optional: override default "shopify" prefix

  - name: personal
    repo: git@github.com:kovyrin/my-prompts.git

config:
  install_path: .ai/prompts
  # List the AI agents you want to export prompts/rules for
  agents:
    - cursor
    - claude

rulesets:
  - shopify/ruby-style
  - shopify/rails-patterns@v3.0
  - shopify/graphql@stable
  - personal/git-workflow
  - personal/testing-utils@v1.0
```
