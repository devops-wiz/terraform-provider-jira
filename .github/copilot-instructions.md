# Terraform Provider for Jira - Development Instructions

**ALWAYS** follow these instructions first and fallback to additional search and context gathering only if the information here is incomplete or found to be in error.

## Working Effectively

### Required Toolchain Setup
Install these exact versions and tools in this order:

```bash
# 1. Verify Go version (REQUIRED: 1.24.x as per go.mod)
go version  # Must be 1.24.x - current: go1.24.6

# 2. Install Task build runner
go install github.com/go-task/task/v3/cmd/task@latest

# 3. Install golangci-lint (REQUIRED: v1.62.0 for config compatibility)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0

# 4. Install Terraform CLI (REQUIRED for docs generation)
# Ubuntu/Debian:
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install -y terraform

# 5. Install GoReleaser v2 (for releases)
go install github.com/goreleaser/goreleaser/v2@latest

# 6. Update PATH to include Go bin
export PATH=$PATH:$(go env GOPATH)/bin
```

### Bootstrap and Build Process
Run these commands in order to set up the development environment:

```bash
# 1. Download dependencies
go mod download
go mod verify

# 2. Build check - takes ~15 seconds, NEVER CANCEL
task build  # Timeout: 60+ seconds

# 3. Generate documentation - takes ~7 seconds, NEVER CANCEL  
task gen    # Timeout: 60+ seconds

# 4. Run linting - takes ~11 seconds, NEVER CANCEL
task lint   # Timeout: 60+ seconds

# 5. Run unit tests - takes ~16 seconds, NEVER CANCEL
task test   # Timeout: 60+ seconds
```

### Critical Timing Information
**NEVER CANCEL these commands - they WILL complete successfully:**

- `task build`: ~15 seconds (basic build check)
- `task test`: ~16 seconds (unit tests only)  
- `task lint`: ~11 seconds (golangci-lint with repo config)
- `task gen`: ~7 seconds (terraform fmt + tfplugindocs)
- `goreleaser release --snapshot --clean`: ~4m47s (multi-arch release build)

**ALWAYS set timeouts of 60+ seconds minimum** for any build command to prevent premature cancellation.

## Running Tests

### Unit Tests (Default)
```bash
# Basic unit tests (skips acceptance by default)
task test

# With additional flags
task test RACE=true COVER=true COVERPROFILE=coverage.out
task test VRBS=true  # Verbose output
task test CTC=true   # Clean test cache
```

### Acceptance Tests (Requires Jira Instance)
**IMPORTANT**: Acceptance tests require a real Jira Cloud instance and credentials.

Setup:
```bash
# 1. Create .env file in repo root (DO NOT commit this file)
cp .env.example .env

# 2. Edit .env with your Jira credentials:
JIRA_ENDPOINT=https://your-org.atlassian.net
JIRA_API_EMAIL=your-email@example.com  
JIRA_API_TOKEN=your-api-token
```

Run acceptance tests:
```bash
# All acceptance tests - takes 2+ minutes, NEVER CANCEL
task test:acc  # Timeout: 300+ seconds

# Filtered acceptance tests
task test ACC_ONLY=true RUN_PATTERN="TestAccWorkTypeResource_basic"

# All tests (unit + acceptance)
task test:all  # Timeout: 300+ seconds
```

### Cleanup Test Artifacts
```bash
task sweep  # Cleans up test resources in Jira (loads .env)
```

## Validation Scenarios

After making changes, **ALWAYS** run these validation steps:

### 1. Code Quality Validation
```bash
# Format and imports
task fmt
task goimports  # Requires: go install golang.org/x/tools/cmd/goimports@latest

# Linting (CRITICAL - CI will fail without this)
task lint

# License headers
task license  # Uses copywrite.hcl config
```

### 2. Build and Documentation Validation  
```bash
# Verify build works
task build

# Regenerate and validate docs (CRITICAL - CI checks for diff)
task gen
git diff  # Should show no changes if docs are current

# Create local release artifacts (optional validation)
goreleaser release --snapshot --clean  # 4m47s, NEVER CANCEL
```

### 3. Manual Provider Testing
```bash
# Build and test provider binary
go build -o terraform-provider-jira
./terraform-provider-jira --help  # Should show usage with --debug flag

# Debug mode for development
./terraform-provider-jira -debug
```

### 4. End-to-End Scenarios
**ALWAYS test at least one complete scenario after making changes:**

1. **Provider Configuration Test**: Use examples/provider/ to verify provider loads correctly
2. **Resource CRUD Test**: If modifying resources, test create/read/update/delete operations  
3. **Data Source Test**: If modifying data sources, verify they return expected data
4. **Documentation Test**: Verify generated docs match code changes

## Pre-commit and CI Preparation

### Pre-commit Hooks (Optional but Recommended)
```bash
# Install pre-commit
pipx install pre-commit  # or: pip install --user pre-commit
pre-commit install

# Run all checks
pre-commit run -a
# OR use Task shortcut:
task precommit
```

### Before Committing - Required Steps
**These steps are MANDATORY** before committing to avoid CI failures:

```bash
# 1. Format and lint (CI will fail if not clean)
task fmt
task goimports  
task lint

# 2. Regenerate docs if schema changed (CI will fail on diff)
task gen
git add docs/  # If any docs were regenerated

# 3. Run license header check
task license

# 4. Verify tests pass
task test  # Unit tests minimum
```

## Project Structure and Key Locations

### Repository Root
```
.
├── CONTRIBUTING.md          # Detailed contributor guide
├── README.md               # Basic project overview  
├── Taskfile.yml            # Build commands and workflows
├── go.mod                  # Go dependencies (requires 1.24.x)
├── main.go                 # Provider entrypoint
├── .golangci.yml           # Linting configuration
├── .goreleaser.yaml        # Release configuration  
├── .pre-commit-config.yaml # Pre-commit hooks
└── copywrite.hcl           # License header config
```

### Core Code
```
internal/provider/          # Main provider implementation
├── provider.go            # Provider schema and configuration
├── provider_config.go     # Configuration validation and setup
├── *_resource.go          # Individual resource implementations
├── *_data_source.go       # Data source implementations
└── testhelpers/           # Test utilities
```

### Documentation and Examples
```
docs/                      # Generated documentation (DO NOT edit manually)
├── index.md              # Provider documentation
├── resources/            # Resource documentation
└── data-sources/         # Data source documentation

examples/                 # Terraform configuration examples
├── provider/             # Provider configuration examples
├── resources/            # Resource usage examples
└── data-sources/         # Data source usage examples

templates/                # Documentation templates for tfplugindocs
```

### Development Files
```
.github/workflows/        # CI/CD workflows
├── ci.yml               # Main CI pipeline
└── release.yml          # Release automation

docs-internal/           # Internal documentation
├── requirements.md      # Project requirements
├── tasks.md            # Development task tracking  
└── plan.md             # Development planning
```

## Common Tasks and Patterns

### Task Runner Commands
```bash
# View all available tasks
task --list

# Build and development
task build                 # Quick build check
task install              # Install provider locally

# Testing  
task test                 # Unit tests only
task test:acc            # Acceptance tests only
task test:all            # All tests

# Code quality
task fmt                 # Go formatting
task goimports           # Import organization
task lint                # Linting with golangci-lint
task license             # Apply license headers

# Documentation
task gen                 # Generate docs (terraform fmt + tfplugindocs) 
task docs                # Alias for task gen

# Releases
task release:snapshot    # Build snapshot artifacts
task release:local       # Build local artifacts only

# Development tools
task delve               # Start delve debug session
task precommit           # Run pre-commit checks
task sweep               # Clean test artifacts
```

### Environment Variables Reference
```bash
# Provider Configuration (for acceptance tests)
JIRA_ENDPOINT=https://your-org.atlassian.net  # or JIRA_BASE_URL
JIRA_API_EMAIL=user@example.com              # or JIRA_EMAIL  
JIRA_API_TOKEN=your-api-token
JIRA_USERNAME=username                        # For basic auth
JIRA_PASSWORD=password                        # For basic auth
JIRA_EMAIL_REDACTION_MODE=mask               # Optional: mask, domain, full

# Test Configuration
TF_ACC=1                                     # Enable acceptance tests
GOFLAGS="-race -cover"                       # Test flags

# Build Configuration  
CGO_ENABLED=0                                # For goreleaser builds
GPG_FINGERPRINT=your-gpg-key                 # For signed releases
```

### Debugging and Development
```bash
# Debug provider with delve
task delve
# In another terminal, run terraform with:
# TF_PROVIDER_ADDR=registry.terraform.io/devops-wiz/jira terraform init

# View build artifacts
ls -la dist/  # After running goreleaser

# Check generated docs
find docs/ -name "*.md" -exec wc -l {} \;
```

## Troubleshooting Common Issues

### Build Failures
- **"golangci-lint version mismatch"**: Install v1.62.0 exactly: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0`
- **"terraform not found"**: Install Terraform CLI for documentation generation
- **"task not found"**: Install Task: `go install github.com/go-task/task/v3/cmd/task@latest`

### Test Failures  
- **Acceptance test connection errors**: Verify JIRA_ENDPOINT, JIRA_API_EMAIL, and JIRA_API_TOKEN in .env file
- **"rate limited"**: Use `terraform apply -parallelism=1` to reduce concurrent requests
- **Timeout issues**: Increase provider timeout settings in examples/provider/retry_and_timeouts/

### Documentation Issues
- **"docs diff in CI"**: Run `task gen` and commit generated files in docs/
- **Missing templates**: Templates are in templates/ directory, regenerated by tfplugindocs

### Release Issues
- **GoReleaser signing errors**: Set GPG_FINGERPRINT environment variable or remove signing config
- **Version compatibility**: Use GoReleaser v2 for version 2 config format

Remember: **NEVER CANCEL** long-running builds or tests. The timings above are tested and accurate.