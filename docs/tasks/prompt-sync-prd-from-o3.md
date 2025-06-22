# Prompt-Sync PRD  
*A package-manager-style tool for installing, version-locking and auditing AI prompt/rule packs across multiple developer agents—without exposing content to public registries.*

---

## 1  Problem

| Pain | Details |
|------|---------|
| **Prompt drift & duplication** | Engineers copy rule files by hand between projects; filenames collide, older versions linger, CI builds diverge. |
| **Security / compliance** | Enterprises can’t adopt community-hosted prompt managers; they need deterministic hashes, audit logs and zero outbound traffic. |
| **Multi-agent friction** | The same prompt often needs to live as a Cursor rule **and** a Claude slash-command (and, later, Copilot, Codeium…). Manual duplication is error-prone. |

---

## 2  Goals & Non-Goals

| Goals | Non-Goals |
|-------|-----------|
| Deterministic installs reproducible in local dev **and** CI. | Public “npm-style” registry on day 1 (can arrive later). |
| Private/org + project + personal overlay model, each with clear precedence. | Fine-grained dependency solver (simple semver ranges are enough). |
| Support at least **Cursor** and **Claude Code** from a single prompt pack. | Web UI or SaaS dashboard (CLI only in v0.x). |
| CI-ready conflict detection (`--strict`). | Interactive three-way merge engine (future). |

---

## 3  Key Concepts

| Term | Meaning |
|------|---------|
| **Prompt Pack** | A Git-versioned folder of markdown prompts/rules + YAML front-matter. |
| **Front-matter Fields** | `title`, `kind` (`rule` \| `command`), `targets` (`cursor`, `claude.cmd`, `claude.md`, `copilot`, …), `scope`, `version`, `security`. |
| **Adapter** | Go plug-in that renders a pack into the on-disk format an agent expects. |
| **Trust Scope** | `org` → `project` → `personal` (highest). Higher scopes shadow lower ones. |
| **Lock File** | `Promptsfile.lock`; maps *pack → adapter → file SHA256 + adapter version* for deterministic builds. |

---

## 4  Architecture

### 4.1 Pack Layout & Metadata

```
my-pack/
├── prompts/
│   └── git.commit.md   # <= contains front-matter
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

*Omitting front-matter ⇒ defaults `kind: rule`, `targets: all`.*

---

### 4.2 Adapter API (Go pseudo-code)

```go
type AgentAdapter interface {
    Name() string            // "cursor", "claude"
    Detect() bool            // auto-enable if agent present
    TargetDir(scope Scope) string
    Render(p PromptPack, scope Scope) ([]RenderedFile, error)
    Verify(files []RenderedFile, mode Strictness) error
}
```

#### Built-in Adapters (v0.1)

| Adapter | Output path | Rules for duplicates |
|---------|-------------|----------------------|
| **Cursor** | `.cursor/rules/` (root) | duplicate *basenames* shadow by scope; `--strict` fails. |
| **Claude (cmd)** | `.claude/commands/` | filename becomes `/scope:command` id. |
| **Claude (md)** | Stitched into `CLAUDE.md` under `## Rules imported from prompt-sync` | duplicate anchor text is an error in `--strict`. |

*Future*: Copilot Chat (`*.prompt.md`), Copilot Instructions, Windsurf, MCP server.

---

### 4.3 Install Flow

1. **Resolve packs** → local Git repos or internal registry (future).
2. **Stage personal / project / org packs** (respect precedence).
3. **Adapter render** → produce files into target dirs.
4. **Conflict scanner**  
   * Within each adapter overlay (e.g. duplicate basenames).  
   * Global cross-scope scan of `**/.cursor/rules/` & `**/.claude/commands/`.
5. **Write lock** (`Promptsfile.lock`) with file hashes for every adapter.
6. **Post-install verify** (`prompt-sync verify`) re-scans workspace for strays.

---

### 4.4 Conflict Handling

| Rule | Dev Mode | `--strict` / CI |
|------|----------|-----------------|
| Same basename after render | Warn, list shadows | Abort install |
| Version-solver clash | Abort | Abort |
| Hash drift vs. lock | Abort | Abort |

Precedence: `personal > project > org`.  
Only **winning** files enter the lock file; personal files never enter shared lock.

---

## 5  CLI Surface

```bash
prompt-sync init                        # create Promptsfile + .gitignore hints
prompt-sync add org/core-git@^1.0
prompt-sync install [--agents=cursor,claude] [--strict]
prompt-sync verify                      # CI hook
prompt-sync publish                     # (future) push pack to private registry
prompt-sync audit                       # (future) JSONL trail for SOC 2
```

---

## 6  Security & Compliance

* **No outbound calls**: packs pulled from internal Git or registry mirrors.  
* **Signed tags (optional v0.2)**: `prompt-sync publish --sign` writes provenance file; install verifies GPG sig.  
* **Prompt-fuzz pre-publish hook (v0.3)**: blocks obviously jail-breakable prompts.  
* **Audit log**: `prompt-sync audit` emits who/when/why for every prompt hash change.

---

## 7  MVP Cut (v0.1)

| Component | Status |
|-----------|--------|
| Go CLI: `init · install · verify` | ✅ |
| Local Git pack resolver | ✅ |
| Cursor + Claude(cmd) adapters | ✅ |
| Duplicate-basename scanner | ✅ |
| Lock file writer / verifier | ✅ |
| Basic docs & example packs | ✅ |

Total estimated effort: **~7 dev-days** (single engineer, ⩽2 h/day focus blocks).

---

## 8  Roadmap (0–6 months)

1. **Private registry alpha** (S3-hosted index, token auth).  
2. **Copilot adapters** – `.prompt.md` + `copilot-instructions.md`.  
3. **Windsurf / Codeium support** – concatenated `rules.md`.  
4. **Prompt-fuzz & policy engine** – OPA/Rego blocklist of critical overrides.  
5. **Signed provenance / SLSA-compliant builds**.  
6. **Interactive resolver** – choose per-file shadow resolution.  
7. **Web status dashboard** (optional): lock drift alerts, pack search.

---

## 9  Open Questions

1. **Binary distribution:** goreleaser + Homebrew, or distribute via internal Artifactory?  
2. **Pack licensing:** MIT vs. org-private?  
3. **Registry format:** static JSON index vs. OCI-style artifact registry?  
4. **Windows symlink strategy:** copy vs. junction?

---

*Prepared June 21 2025*  
*Author: Oleksiy + o3 draft*
