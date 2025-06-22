# Product Requirements Document: prompt-sync

## 1. Introduction/Overview

`prompt-sync` is a package manager for AI assistant prompts, designed to solve the challenge of managing and sharing AI-related files across projects and teams at scale. It handles two distinct types of content:

1. **Persistent Rules/Instructions** - Configuration files that affect AI behavior throughout a project (e.g., Cursor rules, Copilot instructions)
2. **Action Prompts** - Reusable prompts for specific tasks (e.g., generating PRDs, creating test suites, refactoring code)

The tool treats these AI assets like software dependencies - versioned, shareable, and installed locally rather than committed to repositories.

The tool enables developers and organizations to:

- Maintain consistent AI assistant behavior across projects
- Build libraries of reusable action prompts for common tasks
- Share prompt collections within teams and the community
- Layer personal preferences on top of organizational standards
- Keep prompts synchronized and up-to-date
- Avoid repository bloat and merge conflicts

Think of it as "npm for AI prompts" - a familiar package management paradigm applied to the emerging need for AI asset management, but focused on trusted sources only (personal and organizational repositories) to prevent prompt injection attacks.

## 2. Goals

1. **Enable prompt sharing at scale** - Support organizations with thousands of developers sharing common AI prompts
2. **Manage both rules and action prompts** - Handle persistent AI configuration rules and reusable task-specific prompts
3. **Maintain personal flexibility** - Allow developers to layer personal prompt preferences on organizational standards
4. **Treat prompts as dependencies** - Install prompts locally like packages, not commit them as source code
5. **Trust-based security model** - Only support trusted sources (personal and organizational) to prevent prompt injection attacks
6. **Provide reproducible environments** - Lock files ensure consistent prompt versions across team members
7. **Zero learning curve** - Mirror familiar package manager workflows (npm, bundler, pip)
8. **Tool-aware handling** - Special handling for tool-specific formats (e.g., MDC headers for Cursor)

## 3. User Stories

### Core Workflows

1. **New Project Setup**

   - "As a developer joining a new project, I run `prompt-sync install` after cloning and get all the team's standard AI prompts configured in under 30 seconds."

2. **Organizational Standards**

   - "As a Shopify developer, my projects automatically include Shopify's Ruby style guides and Rails patterns for AI assistants, ensuring consistent code suggestions across our 4000+ developers."

3. **Personal Customization**

   - "As a developer with personal preferences, I can layer my own Git workflow prompts and keyboard shortcuts on top of my company's standards without affecting my teammates."

4. **Prompt Discovery**

   - "As a developer starting a React project, I can browse and add prompts from my organization's approved repositories or my personal collection."

5. **Action Prompt Usage**

   - "As a developer, I have a library of action prompts installed that I can reference when needed, like '@generate-prd' to create a PRD or '@create-tests' to generate test suites, without cluttering my project rules."

6. **Version Control**

   - "As a team lead, I can update our shared prompts and know that `prompt-sync update` will distribute changes to all team members consistently."

7. **Multi-Project Management**
   - "As a developer working on multiple projects, each project maintains its own prompt configuration while sharing my personal defaults."

### Administrative Workflows

8. **Prompt Publishing**

   - "As a prompt maintainer, I can publish updates to our organization's prompt repository and developers will receive them on their next update."

9. **Version Pinning**
   - "As a DevOps engineer, I can pin specific prompt versions in CI/CD to ensure reproducible builds."

## 4. Functional Requirements

### 4.1 Configuration Files

#### 4.1.1 Promptsfile (Project-level, committed)

- Defines prompt sources and rulesets for the project
- YAML format for human readability
- Committed to version control
- Shared across all team members

Example:

```yaml
version: 1

sources:
  # IMPORTANT: Only add trusted sources! Untrusted prompts can lead to
  # prompt injection attacks with serious security consequences.

  - name: shopify
    repo: git@github.com:shopify/ai-prompts.git
    # Corporate repository with review process

  - name: personal
    repo: git@github.com:kovyrin/my-prompts.git
    # Your own prompts

config:
  install_path: .ai/prompts # Where prompts are installed
  ai_tool: cursor # Optional: tool-specific settings

rulesets:
  # From shopify source
  - shopify/ruby-style # Tracks default branch (main/master)
  - shopify/rails-patterns@v3.0 # Specific tag
  - shopify/graphql@stable # Track a branch

  # From personal source
  - personal/git-workflow
  - personal/testing-utils@v1.0
```

#### 4.1.2 Promptsfile.lock (Project-level, committed)

- Records exact commits of installed prompts
- Auto-generated, not manually edited
- Ensures reproducible installs
- Committed to version control

Example structure:

```yaml
version: 1
rulesets:
  - name: shopify/ruby-style
    source: shopify
    requested_ref: v2.1.0 # What was requested (tag/branch)
    resolved_ref: abc123def456 # Actual commit SHA
    updated_at: 2025-01-20T10:30:00Z
```

#### 4.1.3 Promptsfile.local (Project-level, git-ignored)

- Personal project-specific additions
- Overrides and additions to Promptsfile
- Never committed
- Optional file

#### 4.1.4 ~/.prompt-sync/config.yaml (User-level)

- Global user configuration
- Personal prompt sources
- Default rulesets applied to all projects
- Local paths to cloned repositories

### 4.2 Core Commands

#### 4.2.1 `prompt-sync init`

- Initializes a new project with prompt management
- Creates a basic Promptsfile with intelligent defaults
- Interactive mode to select from common configurations
- Detects project type and suggests relevant rulesets

#### 4.2.2 `prompt-sync install`

- Installs all prompts defined in configuration files
- Reads from Promptsfile, user config, and local overrides
- Uses Promptsfile.lock if present for exact versions
- Creates .ai/prompts/ directory structure
- Clones or updates git repositories as needed

#### 4.2.3 `prompt-sync add <source/ruleset>`

- Adds a new ruleset to the project
- Updates Promptsfile and Promptsfile.lock
- Supports various formats:
  - `prompt-sync add shopify/ruby-style` - Track default branch
  - `prompt-sync add shopify/ruby-style@v1.0` - Use specific tag
  - `prompt-sync add shopify/ruby-style@stable` - Track branch
  - `prompt-sync add github:username/repo/ruleset` - From GitHub URL

#### 4.2.4 `prompt-sync update [ruleset]`

- Updates prompts to latest commits on tracked branches
- Without arguments, updates all rulesets
- With ruleset specified, updates only that ruleset
- Tagged/commit references are not updated (pinned)
- Updates Promptsfile.lock with new commit SHAs

#### 4.2.5 `prompt-sync remove <ruleset>`

- Removes a ruleset from the project
- Updates configuration files
- Cleans up installed files

#### 4.2.6 `prompt-sync list`

- Shows installed rulesets and their versions
- Can show available rulesets from configured sources
- Options:
  - `--source=shopify` - List from specific source
  - `--outdated` - Show rulesets with updates available
  - `--files` - Show individual files in rulesets

#### 4.2.7 `prompt-sync search <query>`

- Search for rulesets within cloned repositories
- Searches metadata.yaml files for matches
- Limited to already-configured sources (no global search)
- Shows matching rulesets with descriptions

#### 4.2.8 `prompt-sync publish [ruleset]`

- Copies local changes to the appropriate git repository
- Creates a commit with descriptive message
- Attempts to push based on repository type:
  - **Personal repos**: Push directly to main (default)
  - **Corporate repos**: Create feature branch and push there
- Handles push failures gracefully:
  - If direct push fails, offer to create a branch
  - Provide clear instructions for creating a PR
  - Never leave the repository in a broken state
- Options:
  - `--branch=feature-name` - Explicitly use a branch
  - `--message="commit message"` - Custom commit message
- Uses git's configured user for commit attribution

#### 4.2.9 `prompt-sync import <url>`

- Import prompts from a repository into your personal collection
- Requires explicit confirmation with security warning
- Supports formats:
  - `prompt-sync import github:username/repo/ruleset`
  - `prompt-sync import https://github.com/user/repo`
- Imports into your personal repository for review before use
- Shows clear warning about security risks of untrusted sources

### 4.3 Directory Structure

```
project-root/
├── Promptsfile              # Project configuration (committed)
├── Promptsfile.lock         # Lock file (committed)
├── Promptsfile.local        # Personal overrides (git-ignored)
├── .gitignore              # Includes .ai/prompts/
└── .ai/
    └── prompts/            # Installed prompts (git-ignored)
        ├── shopify/        # Organizational prompts
        │   ├── ruby-style/
        │   │   ├── testing.md
        │   │   └── style-guide.md
        │   └── rails-patterns/
        └── personal/       # Your personal prompts
            ├── git-workflow/
            └── testing-utils/
```

### 4.4 Ruleset Structure

Each ruleset in a source repository can contain both persistent rules and action prompts:

```
ruleset-name/
├── metadata.yaml           # Ruleset metadata
├── rules/                  # Persistent AI behavior rules
│   ├── coding-style.mdc   # Cursor rule with header
│   └── git-workflow.md    # General rule
├── prompts/               # Reusable action prompts
│   ├── generate-prd.mdc   # PRD generation prompt
│   ├── create-tests.md    # Test generation prompt
│   └── refactor/          # Organized by category
│       └── extract-function.md
└── shared/                # Can be used as either
    └── documentation.md
```

metadata.yaml format:

```yaml
name: "Ruby Development Kit"
description: "Shopify's Ruby rules and prompts for AI assistants"
author: "Shopify"
tags: ["ruby", "style-guide", "testing", "prompts"]
content_types:
  - rules # Has persistent rules
  - prompts # Has action prompts
dependencies: # Other rulesets this depends on
  - shopify/general-patterns
  - shopify/testing-utils@v2.0 # Can specify tag/branch
min_prompt_sync_version: "1.0.0"
```

### 4.5 Version Management

Git-based versioning provides flexibility and simplicity:

- **Default behavior**: Track the repository's default branch (main/master)
- **Reference formats** in Promptsfile:
  - `shopify/ruby-style` - Track default branch
  - `shopify/ruby-style@v2.1.0` - Use a specific tag
  - `shopify/ruby-style@stable` - Track a specific branch
  - `shopify/ruby-style@abc123d` - Pin to exact commit
- **Lock file**: Always records the exact commit SHA for reproducibility
- **Updates**: `prompt-sync update` pulls latest commits from tracked branches
- **Tagging strategy**: Ruleset authors can use any tagging scheme (or none)

### 4.6 Authentication & Security

- Relies entirely on the user's existing git configuration
- No credential management needed - git handles SSH keys, tokens, etc.
- Works with any authentication method the user has configured for git
- Supports private repositories through standard git authentication
- No credentials stored or managed by prompt-sync itself

### 4.7 Conflict Resolution

When multiple sources provide the same file:

1. Priority order: local > personal > project > organization
2. Clear reporting of conflicts
3. Option to override with explicit source selection
4. Maintain attribution for debugging

#### 4.7.1 Special Handling for MDC Files

Cursor's MDC files require special conflict detection due to their header structure:

```mdc
---
scope: project  # User might change this per project
tools: true
---

# Actual content starts here
```

For MDC files specifically:

- Headers are **excluded** from conflict detection
- Only the content after the header is compared
- Users can modify scope/tools settings without creating conflicts
- Content comparison ignores whitespace differences
- Checksums/hashes cannot be used alone due to header variations

### 4.8 Performance Requirements

- Install 100 prompt files in < 5 seconds (after initial clone)
- Git operations run sequentially (git's own parallelism applies)
- Initial clone time depends on repository size and network
- Minimal overhead on top of git operations
- Quick startup time (< 200ms for simple commands)

## 5. Non-Goals (Out of Scope for MVP)

1. **GUI or IDE integration** - Command-line only initially
2. **Automatic prompt generation** - No AI-generated prompts
3. **Prompt effectiveness analytics** - No tracking of prompt usage
4. **Built-in prompt editor** - Use external editors
5. **Conflict resolution UI** - Simple priority rules only
6. **Community features** - Explicitly excluded due to security risks
7. **Paid/premium prompts** - No monetization features
8. **Windows support** - Focus on macOS/Linux initially

## 6. Design Considerations

### 6.1 User Interface

- Clean, informative CLI output with progress indicators
- Color-coded output for different types of messages
- Verbose and quiet modes
- JSON output option for scripting
- Clear error messages with actionable next steps

### 6.2 Migration Path

For users migrating from ai-rule-sync or manual management:

- `prompt-sync migrate` command to convert existing setups
- Detect existing prompt directories and offer to import
- Preserve Git history when importing

### 6.3 Tool Agnostic

- Work with any AI tool's configuration format
- Configurable file extensions and directory names
- Support for tool-specific subdirectories

### 6.4 Git Workflow Agnostic

- No assumptions about git workflow (PR vs direct push)
- Graceful handling of permission failures
- Smart defaults with manual overrides
- Let git permissions drive the workflow:
  - Try direct push first (simplest case)
  - On failure, guide through branch creation
  - Never force users into a specific workflow
- Clear error messages that explain next steps

## 7. Technical Considerations

### 7.1 Implementation Language

- **Go** - Single binary distribution, excellent CLI libraries
- Statically compiled for easy installation
- Cross-platform support (macOS, Linux, WSL)

### 7.2 Dependencies

- Minimal external dependencies
- Git operations via shelling out to git command (simpler, leverages user's git config)
- YAML parsing with gopkg.in/yaml.v3
- Cobra for CLI framework
- Viper for configuration management
- Requires git to be installed on the system

### 7.3 Distribution

- Single binary via GitHub releases
- Homebrew formula for macOS/Linux
- Installation script: `curl -sSL https://prompt-sync.dev/install | bash`
- Eventually: Native package managers (apt, yum, etc.)

### 7.4 Storage

- Git repositories cloned to `~/.prompt-sync/repos/`
- Git handles version storage and history
- Each source repository cloned once and updated as needed
- Offline support through local git clones
- No separate caching layer - git repos ARE the cache

### 7.5 Compatibility

- Git 2.0+ required
- Works with any Git hosting (GitHub, GitLab, Bitbucket)
- Support for SSH and HTTPS protocols
- Compatible with corporate proxies

## 8. Security Considerations

### 8.1 Prompt Injection Risks

- **No untrusted sources**: The tool explicitly does NOT support community or untrusted repositories
- **Trust model**: Only personal repos and organizational repos with proper access controls
- **Why**: AI assistants often have broad system access (file system, terminal, etc.)
- **Risk**: Malicious prompts could exfiltrate data, delete files, or execute harmful commands
- **Mitigation**: Require explicit trust relationship (personal ownership or corporate governance)

### 8.2 Security Best Practices

- Only add sources you control or your organization controls
- Review prompts before installing, especially after updates
- Use branch protection and PR reviews for organizational repositories
- Consider security scanning for prompt repositories in enterprise settings

## 9. Success Metrics

### 9.1 Adoption Metrics (First 6 months)

- 1,000+ developers actively using the tool
- 50+ organizational prompt repositories created
- 500+ personal prompt repositories in use
- 10,000+ project installs

### 9.2 Quality Metrics

- Zero data loss incidents
- Zero security incidents from prompt injection
- < 0.1% failed installs
- 95% of commands complete in < 5 seconds
- 90% user satisfaction in surveys

### 9.3 Ecosystem Metrics

- Average 5+ rulesets per project
- 50% of users maintaining personal prompt repos
- Strong adoption in enterprise environments
- Integration with major AI tools

## 10. Open Questions

1. **Prompt Verification** - Should we add any verification/signing mechanism for organizational prompts?

2. **Monetization** - How do we sustain development? Enterprise features, support contracts, or purely open source?

3. **Security Scanning** - Should we build prompt security scanning features to detect potential injection attempts?

4. **AI Tool Integration** - Should we build direct integrations with Cursor, Continue, Copilot, etc.?

5. **Semantic Analysis** - Should we analyze prompts for conflicts or redundancy?

6. **Licensing** - What license model for shared prompts? How to handle attribution?

7. **Git Integration Depth** - Should we shell out to git or use a Git library? Trade-offs between simplicity and features.

8. **PR Creation** - Should the tool help create PRs after pushing branches, or just provide the URL? Integration with gh CLI?

## 10. Implementation Phases

### Phase 1: Core Package Management (MVP)

- Basic Promptsfile parsing
- Install/update/remove commands
- Git repository support
- Lock file generation
- Multiple source support

### Phase 2: Advanced Features

- Dependency resolution
- Conflict detection (including MDC header handling)
- Search functionality
- Import from external sources (with security warnings)
- Branch/tag tracking improvements

### Phase 3: Enterprise & Security

- Security scanning for prompts
- Audit logging for prompt usage
- Prompt signing/verification
- Enterprise onboarding tools
- Integration with corporate security tools
- Compliance reporting
- Support for git submodules or subtrees

### Phase 4: Advanced Features

- AI tool-specific integrations
- Advanced conflict resolution
- Prompt effectiveness analytics (privacy-preserving)
- Multi-language documentation
- Plugin system for extensions

## 11. Example Scenarios

### 11.1 Individual Developer

```bash
# Start a new Go project
$ prompt-sync init
? Select project type: Go
? Add personal prompts? Yes
? Add organizational prompts? Yes (shopify)
Created Promptsfile with trusted sources only

$ prompt-sync install
Installing rulesets...
  ✓ personal/defaults (from ~/.prompt-sync)
  ✓ shopify/go-style (from organizational repo)
  ✓ personal/git-workflow

$ prompt-sync add personal/go-testing
$ git add Promptsfile Promptsfile.lock
$ git commit -m "Add Go testing prompts"
```

### 11.2 Enterprise Team

```bash
# Shopify developer clones a project
$ git clone shopify/online-store
$ cd online-store
$ prompt-sync install
Installing rulesets...
  ✓ shopify/ruby-style@v2.1.0 (tag: v2.1.0, commit: abc123d)
  ✓ shopify/rails-patterns (branch: main, commit: def456e)
  ✓ shopify/graphql@stable (branch: stable, commit: 789ghi0)
  ✓ personal/shortcuts (from ~/.prompt-sync)

$ prompt-sync list
Installed rulesets:
  shopify/ruby-style      @v2.1.0   abc123d  (4 files)
  shopify/rails-patterns  @main     def456e  (12 files)
  shopify/graphql         @stable   789ghi0  (6 files)
  personal/shortcuts      @main     latest   (2 files)
```

### 11.3 Prompt Author

```bash
# Create and share a new ruleset with both rules and action prompts
$ cd ~/my-prompts
$ mkdir -p frameworks/nextjs/{rules,prompts}

# Add a persistent rule
$ cat > frameworks/nextjs/rules/routing.mdc << EOF
---
scope: project
---
# NextJS Routing Best Practices
Always use the app directory structure...
EOF

# Add an action prompt
$ cat > frameworks/nextjs/prompts/create-api-route.mdc << EOF
---
scope: selection
---
# Create NextJS API Route
Generate a new API route handler with proper TypeScript types...
EOF

$ prompt-sync publish frameworks/nextjs
Published frameworks/nextjs to personal/my-prompts (pushed to main)

# In another project
$ prompt-sync add kovyrin/my-prompts/frameworks/nextjs
```

### 11.5 Security Warning Example

```bash
# User tries to add an unknown source
$ prompt-sync add https://github.com/random-user/cool-prompts

WARNING: You're about to add prompts from an untrusted source.

AI assistants with tool access can execute code on your system.
Malicious prompts could:
- Delete or modify your files
- Access sensitive data
- Execute harmful commands

Only add sources you trust completely.

Do you own or trust this repository? (yes/no): no
Aborted: Not adding untrusted source.
```

### 11.4 Corporate Publish Workflow

```bash
# Developer wants to contribute back to corporate rules
$ prompt-sync publish shopify/ruby-style

Attempting to push to main...
Error: Push to main rejected (protected branch)

Would you like to:
1. Create a feature branch for a pull request
2. Cancel

> 1

Enter branch name (or press Enter for 'update-ruby-style-20250120'): improve-testing-rules

Creating branch 'improve-testing-rules'...
Pushing to origin/improve-testing-rules...
Success!

To create a pull request, visit:
https://github.com/shopify/ai-prompts/compare/improve-testing-rules

# Alternative: specify branch upfront
$ prompt-sync publish shopify/ruby-style --branch=my-updates
```

## 12. Conclusion

`prompt-sync` represents a paradigm shift in how we manage AI assistant configurations. By treating prompts as dependencies rather than source code, we enable scalable, maintainable, and shareable AI assistance across projects and organizations.

The tool's security-first approach, limiting sources to trusted personal and organizational repositories, protects users from prompt injection attacks while still enabling powerful sharing workflows. This balance between functionality and security makes it suitable for both individual developers and large enterprises.

The familiar package manager model ensures rapid adoption while solving real problems faced by development teams today.

---

_Last updated: 2025-01-20_
