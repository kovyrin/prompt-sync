#!/bin/bash
# Script to set up comprehensive test fixtures for prompt-sync

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPOS_DIR="$SCRIPT_DIR/repos"

echo "Setting up test fixtures in $REPOS_DIR..."

# Function to create a git repo with initial commit
create_repo() {
    local repo_name=$1
    local repo_path="$REPOS_DIR/$repo_name"

    echo "Creating $repo_name..."
    rm -rf "$repo_path"
    mkdir -p "$repo_path"
    cd "$repo_path"

    git init --quiet
    git config user.email "test@example.com"
    git config user.name "Test User"
}

# Function to commit changes
commit_changes() {
    local message=$1
    git add -A
    git commit -m "$message" --quiet
}

# 1. Enterprise prompts repo with MDC frontmatter and metadata
create_repo "enterprise-prompts"

mkdir -p prompts/architecture prompts/security prompts/testing

# MDC file with comprehensive frontmatter
cat > prompts/architecture/microservices.mdc << 'EOF'
---
title: Microservices Architecture Guidelines
description: Best practices for designing and implementing microservices
tags:
  - architecture
  - microservices
  - distributed-systems
metadata:
  version: "2.0"
  author: "Architecture Team"
  last_updated: "2024-01-15"
  complexity: advanced
  estimated_time: "45 minutes"
settings:
  strict_mode: true
  require_review: true
  min_experience_level: senior
---

# Microservices Architecture Guidelines

This guide provides comprehensive patterns and practices for microservices.

::alert{type="warning"}
These patterns require strong understanding of distributed systems.
::

## Core Principles

1. **Service Independence**: Each service should be independently deployable
2. **Domain Driven Design**: Services should align with business domains
3. **API First**: Design APIs before implementation

::code-group
```yaml [docker-compose.yml]
version: '3.8'
services:
  api-gateway:
    image: nginx:alpine
    ports:
      - "80:80"
```

```typescript [service.ts]
interface ServiceConfig {
  name: string;
  port: number;
  dependencies: string[];
}
```
::

## Best Practices

::checklist
- Use API versioning
- Implement circuit breakers
- Add comprehensive logging
- Monitor service health
::
EOF

# Regular markdown with YAML frontmatter
cat > prompts/security/authentication.md << 'EOF'
---
title: Authentication Best Practices
priority: critical
applies_to:
  - backend
  - frontend
  - mobile
---

# Authentication Best Practices

Implement secure authentication following OWASP guidelines.

## Key Requirements

- Use secure password hashing (bcrypt, argon2)
- Implement MFA where possible
- Use secure session management
- Implement rate limiting
EOF

# Simple prompt without frontmatter
cat > prompts/testing/unit-testing.md << 'EOF'
# Unit Testing Guidelines

Write comprehensive unit tests for all business logic.

- Aim for 80%+ code coverage
- Use descriptive test names
- Follow AAA pattern (Arrange, Act, Assert)
- Mock external dependencies
EOF

commit_changes "Initial enterprise prompts with MDC and metadata"

# Create multiple versions and tags
git tag -a v1.0.0 -m "Initial release"

# Add more content for v1.1.0
cat > prompts/architecture/event-driven.md << 'EOF'
# Event-Driven Architecture

Implement loosely coupled systems using events.

- Use message queues for async communication
- Implement event sourcing where appropriate
- Design for eventual consistency
EOF

commit_changes "Add event-driven architecture guide"
git tag -a v1.1.0 -m "Add event-driven patterns"

# Add content for v2.0.0 (breaking changes)
cat > prompts/breaking-changes.md << 'EOF'
# Breaking Changes in v2.0.0

- Renamed authentication.md to auth-patterns.md
- Updated all MDC components to latest syntax
- Removed deprecated security patterns
EOF

mv prompts/security/authentication.md prompts/security/auth-patterns.md
commit_changes "Breaking changes for v2.0.0"
git tag -a v2.0.0 -m "Major version with breaking changes"

# Create feature branch
git checkout -b feature/experimental-patterns
mkdir -p prompts/experimental
cat > prompts/experimental/quantum-ready.md << 'EOF'
# Quantum-Ready Cryptography

Experimental patterns for quantum-resistant encryption.
EOF
commit_changes "Add experimental quantum patterns"

git checkout master

# 2. Team prompts with various structures
create_repo "team-standards"

# Nested structure
mkdir -p rules/backend/golang rules/backend/python rules/frontend/react rules/frontend/vue
mkdir -p templates/pr templates/issues templates/rfcs

cat > rules/backend/golang/style.md << 'EOF'
# Go Style Guide

Follow the official Go style guide with team additions:
- Use table-driven tests
- Prefer explicit error handling
- Use context for cancellation
EOF

cat > rules/backend/python/style.md << 'EOF'
# Python Style Guide

Follow PEP 8 with team conventions:
- Use type hints for all functions
- Prefer dataclasses over dictionaries
- Use black for formatting
EOF

cat > rules/frontend/react/components.md << 'EOF'
# React Component Guidelines

- Prefer functional components
- Use TypeScript strict mode
- Implement proper error boundaries
EOF

cat > templates/pr/template.md << 'EOF'
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
EOF

commit_changes "Initial team standards"
git tag -a v1.0.0 -m "Initial team standards"

# Create develop branch with newer content
git checkout -b develop
mkdir -p rules/ai
cat > rules/ai/prompting.md << 'EOF'
# AI Prompting Guidelines

Best practices for AI interactions:
- Be specific and clear
- Provide context
- Use examples when needed
EOF
commit_changes "Add AI prompting guidelines"

git checkout master

# 3. Personal productivity prompts
create_repo "personal-productivity"

mkdir -p prompts/daily prompts/weekly prompts/planning

cat > prompts/daily/standup.md << 'EOF'
# Daily Standup Template

## Yesterday
- What did I complete?

## Today
- What will I work on?

## Blockers
- Any impediments?
EOF

cat > prompts/weekly/review.md << 'EOF'
# Weekly Review

## Accomplishments
- Major wins this week

## Learnings
- What did I learn?

## Next Week
- Key priorities
EOF

commit_changes "Personal productivity templates"

# Add some bulk content to test performance
echo "Creating bulk content for performance testing..."
for i in {1..50}; do
    cat > "prompts/planning/project-${i}.md" << EOF
# Project $i Planning

Generic planning template for project $i.
- Objective: Complete feature $i
- Timeline: Sprint $i
- Resources: Team $i
EOF
done

commit_changes "Add bulk planning templates"
git tag -a v1.0.0 -m "Version with bulk content"

# 4. Multi-language documentation repo
create_repo "multi-language-docs"

mkdir -p docs/en docs/es docs/fr docs/ja

cat > docs/en/coding.md << 'EOF'
# Coding Standards

Write clean, maintainable code.
EOF

cat > docs/es/coding.md << 'EOF'
# Estándares de Codificación

Escribe código limpio y mantenible.
EOF

cat > docs/fr/coding.md << 'EOF'
# Normes de Codage

Écrivez du code propre et maintenable.
EOF

cat > docs/ja/coding.md << 'EOF'
# コーディング標準

クリーンで保守可能なコードを書く。
EOF

commit_changes "Multi-language documentation"

# 5. Conflicting prompts repo (for testing conflict resolution)
create_repo "conflicting-prompts"

mkdir -p prompts

# Create files that conflict with other repos
cat > prompts/coding.md << 'EOF'
# Coding Standards (Conflicting Version)

This conflicts with acme-prompts/prompts/coding.md
EOF

cat > prompts/testing.md << 'EOF'
# Testing Standards (Conflicting Version)

This conflicts with dev-prompts/prompts/testing.md
EOF

commit_changes "Prompts that conflict with other repos"

# Return to original directory
cd "$SCRIPT_DIR"

echo "Test fixtures created successfully!"
echo ""
echo "Created repositories:"
echo "  - enterprise-prompts: MDC files, frontmatter, multiple versions (v1.0.0, v1.1.0, v2.0.0)"
echo "  - team-standards: Nested structure, multiple branches (master, develop)"
echo "  - personal-productivity: 50+ files for performance testing"
echo "  - multi-language-docs: Internationalization example"
echo "  - conflicting-prompts: Files that conflict with existing repos"
