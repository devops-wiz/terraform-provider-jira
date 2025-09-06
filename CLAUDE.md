# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Required Toolchain

Install these exact versions in order:
```bash
# Go 1.24.x (required as per go.mod)
go version  # Must be 1.24.x

# Task build runner (required for all commands)
go install github.com/go-task/task/v3/cmd/task@latest

# golangci-lint v1.62.0 (exact version required)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0

# Terraform CLI (required for docs generation)
# Install via package manager for your OS

# Update PATH
export PATH=$PATH:$(go env GOPATH)/bin
```

## Essential Commands

### Build and Validation (Run these before committing)
```bash
task build    # Build check (~15 seconds)
task lint     # Linting (~11 seconds)  
task gen      # Generate docs (~7 seconds)
task test     # Unit tests (~16 seconds)
```

**IMPORTANT**: Never cancel these commands - they will complete. Set timeouts of 60+ seconds minimum.

### Testing
```bash
# Unit tests (default)
task test

# Acceptance tests (requires Jira instance - see .env.example)
task test:acc  # Timeout: 300+ seconds

# Single test
task test RUN_PATTERN="TestAccWorkTypeResource_basic"
```

### Before Committing (MANDATORY)
```bash
task fmt         # Format code
task goimports   # Fix imports
task lint        # Must pass for CI
task gen         # Regenerate docs if schema changed
task license     # Apply license headers
task test        # Verify tests pass
```

## Architecture Overview

### Provider Structure
The provider uses Terraform Plugin Framework and the go-atlassian library for Jira API interactions.

```
internal/provider/
├── provider.go              # Main provider configuration and schema
├── provider_config.go       # Configuration validation, client setup, retry logic
├── crud_runner.go          # Generic CRUD operations framework for resources
├── *_resource.go           # Resource implementations (CRUD operations)
├── *_data_source.go        # Data source implementations (read-only)
└── testhelpers/            # Test utilities and fixtures
```

### Key Design Patterns

1. **CRUD Runner Pattern**: Resources use a generic `CRUDRunner` that handles:
   - Standard CRUD operations with retry logic
   - State management and drift detection
   - Error handling and logging
   - Timeout configuration

2. **Configuration Hierarchy**: 
   - Provider-level settings (authentication, timeouts, retry)
   - Resource-level overrides via `operation_timeouts` blocks
   - Environment variable fallbacks

3. **Authentication Methods**:
   - API Token (preferred for Jira Cloud)
   - Basic Auth (username/password)
   - Personal Access Token

### Resource Implementation Pattern
All resources follow this structure:
1. Define schema with validators
2. Implement CRUD methods using `CRUDRunner`
3. Map between Terraform state and Jira API models
4. Handle computed fields and defaults
5. Provide import functionality

## Testing Strategy

### Acceptance Tests
- Require real Jira instance (configure via .env file)
- Test full CRUD lifecycle
- Verify drift detection and correction
- Located in `*_test.go` files with `TestAcc` prefix

### Environment Setup for Testing
```bash
# Create .env file (DO NOT commit)
cp .env.example .env

# Configure with your Jira credentials:
JIRA_ENDPOINT=https://your-org.atlassian.net
JIRA_API_EMAIL=your-email@example.com
JIRA_API_TOKEN=your-api-token
```

## Documentation Generation

Documentation is auto-generated using tfplugindocs:
- Templates in `templates/` directory
- Generated docs in `docs/` directory (DO NOT edit manually)
- Examples in `examples/` directory feed into docs
- Run `task gen` to regenerate after schema changes

## Common Development Workflows

### Adding a New Resource
1. Create `internal/provider/<resource>_resource.go`
2. Define schema with appropriate validators
3. Implement CRUD operations using `CRUDRunner`
4. Add acceptance tests in `<resource>_resource_test.go`
5. Create example in `examples/resources/<resource>/`
6. Run `task gen` to generate documentation
7. Run full validation: `task fmt && task lint && task test && task gen`

### Modifying Existing Resources
1. Update schema in resource file
2. Update CRUD logic if needed
3. Update or add tests
4. Update example if schema changed
5. Run `task gen` to regenerate docs
6. Verify no breaking changes

### Debugging Provider
```bash
# Start debug session
task delve

# In another terminal, run terraform with:
TF_PROVIDER_ADDR=registry.terraform.io/devops-wiz/jira terraform init
```

## CI/CD Requirements

The CI pipeline enforces:
- Go 1.24.x version
- All linting rules pass (golangci-lint)
- Documentation is up-to-date (no diff after `task gen`)
- All unit tests pass
- License headers present (MPL-2.0)

## Important Notes

- This is a Terraform provider for Jira using the Terraform Plugin Framework
- The provider interacts with Jira Cloud/Server/Data Center via go-atlassian library
- All resources support import functionality
- Provider implements comprehensive retry logic for API rate limiting
- Operation timeouts are configurable at both provider and resource levels
- The codebase follows Mozilla Public License 2.0