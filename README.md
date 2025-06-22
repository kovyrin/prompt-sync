# Prompt-Sync

> 🛠️ **Package-manager-style CLI for trusted AI prompts & rules**
>
> Think **"npm for AI prompts"** – install, version-lock and audit prompt packs locally **without public registries**.

---

## ✨ Why Prompt-Sync?

Modern developer agents (Cursor, Claude, Copilot, …) rely on markdown prompts and rule files sprinkled across repos and laptops. Copy-pasting them leads to drift, duplication, and security blind-spots. Prompt-Sync applies proven package-management principles so teams can:

- 📦 **Install** prompt packs from ordinary Git repositories
- 🔒 **Version-lock** & reproduce installs in CI via a lock-file
- 🕵️ **Audit** every prompt change in pull-requests
- 🤖 **Render** the same logical prompt for multiple agents automatically
- 🔐 **Stay offline-friendly** – zero outbound traffic by default

See [`docs/tasks/prompt-sync-mvp-prd.md`](docs/tasks/prompt-sync-mvp-prd.md) for the full Product Requirements Document.

---

## 🚀 Quick Start

```bash
# 1. Install the CLI (Homebrew tap coming soon)
go install github.com/kovyrin/prompt-sync/cmd/prompt-sync@latest

# 2. Initialise a new repo
prompt-sync init  # generates Promptsfile & .gitignore block

# 3. Add a prompt pack
prompt-sync add shopify/graphql@stable

# 4. Install / render prompts for Cursor & Claude
prompt-sync install   # deterministic, uses Promptsfile.lock
```

Running in CI? Use the non-interactive wrapper:

```bash
prompt-sync ci-install  # alias for `install --yes --strict`
```

> **Tip:** All commands honour `CI=true` and exit non-zero when user input would be required.

---

## 🔑 Core Concepts

- **Prompt Pack** – Git-versioned directory of markdown prompts plus YAML metadata
- **Adapter** – Plug-in that renders a pack into agent-specific files (e.g., Cursor rules, Claude slash-commands)
- **Scope / Trust Level** – `org → project → personal`; higher scopes shadow lower ones
- **Promptsfile** – Declarative manifest committed to the repo (similar to `package.json`)
- **Promptsfile.lock** – Auto-generated file pinning commit SHAs and file hashes (similar to `package-lock.json`)

---

## 🛠️ CLI Overview (v0.1)

- `prompt-sync init` – Scaffold `Promptsfile`, local config, and managed `.gitignore` entries
- `prompt-sync install [--agents=cursor,claude] [--strict]` – Resolve packs, render via adapters, and update the lock file
- `prompt-sync add <source/pack[@ref]>` – Add a new pack to the manifest and lock file
- `prompt-sync update [<pack>]` – Pull latest commits on tracked branches
- `prompt-sync remove <pack>` – Remove a pack and clean rendered files
- `prompt-sync list [--outdated] [--files]` – Show installed packs and versions
- `prompt-sync verify` – Re-render in CI and fail on drift

Run any command with `--help` for detailed flags.

---

## 🗂️ Configuration Files

```text
project-root/
├── Promptsfile          # committed – trusted sources & selected packs
├── Promptsfile.lock     # committed – resolved SHAs & hashes
├── Promptsfile.local    # git-ignored – per-developer overrides
└── ~/.prompt-sync/config.yaml  # user-level defaults
```

Example minimal **Promptsfile**:

```yaml
version: 1

sources:
  - name: shopify
    repo: git@github.com:shopify/ai-prompts.git

rulesets:
  - shopify/graphql@stable
```

---

## 🛡️ Security Model

1. **Trusted Sources Only** – Repos must be allow-listed in a `sources:` block; unknown remotes cause an error (or prompt with `--allow-unknown`).
2. **Zero Credential Storage** – Prompt-Sync defers to your existing Git SSH keys / tokens.
3. **Security Levels** – Each prompt can declare `security: low|medium|high`; CI can block risky prompts in `--strict` mode.
4. **Deterministic Builds** – The lock file pins **both** commit SHAs and file hashes; drift detection fails CI.

---

## 🏗️ Roadmap

The MVP (`v0.1`) delivers the core package-management flow, Cursor & Claude adapters, and CI verification. Planned next milestones include:

- Copilot & Codeium adapters
- Offline / air-gapped registry mirroring
- Advanced conflict-resolution UI

Track progress in the [project roadmap](docs/tasks/prompt-sync-mvp-prd.md#9-roadmap).

---

## 🤝 Contributing

Pull-requests and issues are welcome! Please:

1. Read the [code of conduct](CODE_OF_CONDUCT.md) (TBD).
2. Open an issue to discuss major changes.
3. Ensure `go test ./...` & `prompt-sync verify` pass.

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for details.

---

### Maintainer

Oleksiy Kovyrin – [oleksiy@kovyrin.net](mailto:oleksiy@kovyrin.net)
