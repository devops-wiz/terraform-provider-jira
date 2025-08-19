# Terraform Provider for Jira

A Terraform provider for managing Jira resources using the [go-atlassian](https://github.com/ctreminiom/go-atlassian) library.

[![Go Report Card](https://goreportcard.com/badge/github.com/devops-wiz/terraform-provider-jira)](https://goreportcard.com/report/github.com/devops-wiz/terraform-provider-jira)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) 1.24.x (tested in CI)

## Installation

### Terraform Registry (Recommended)

The provider is available on the [Terraform Registry](https://registry.terraform.io/providers/devops-wiz/jira/latest).

```hcl
terraform {
  required_providers {
    jira = {
      source  = "devops-wiz/jira"
      version = "~> 1.0"
    }
  }
}

provider "jira" {
  # Configuration options
}
```

### Local Build

1. Clone the repository
2. Build the provider using `go build -o terraform-provider-jira`
3. Move the binary to the appropriate Terraform plugin directory

## Contributing

Please read our [CONTRIBUTING.md](CONTRIBUTING.md) for environment setup, build, lint, test, and docs workflows. We standardize on Go 1.24.x (go.mod toolchain set to 1.24.3). Optional local hooks are available via [pre-commit](https://pre-commit.com).

Community and Security:

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)

We welcome PRs! Keep changes focused, include tests where possible, and update docs as needed. See CONTRIBUTING for acceptance test guidance.

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgements

- [go-atlassian](https://github.com/ctreminiom/go-atlassian) - The Go library used to interact with Atlassian APIs
- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) - The framework used to develop this provider


## JetBrains HTTP Client requests (API testing)

Use the curated HTTP requests under docs-internal/http/ to explore and verify Jira REST API behavior with the JetBrains HTTP Client.

- Files: docs-internal/http/*.http (MVP and v1/future groups; split files are the source of truth)
- Environment:
  1) Copy http-client.env.json.example to http-client.private.env.json (preferred) at the repo root.
  2) Fill values:
     - JIRA_BASE_URL (e.g., https://your-org.atlassian.net)
     - JIRA_EMAIL
     - JIRA_API_TOKEN
     - JIRA_BASIC_AUTH (base64 of "email:api-token"). See examples inside the example file.
  3) In your JetBrains IDE, select the environment (e.g., dev) in the HTTP Client toolbar.
- Run: Open any .http file and click the gutter run icons to execute requests.
- Notes:
  - Test against a non-production Jira site. Many requests create/update/delete admin entities.
  - Some list endpoints are non-paginated and heavy (e.g., GET /rest/api/3/field). Prefer paginated alternatives where available.
  - Some examples include optional query params like expand (e.g., statuses expand=usages, projects search expand options) and delete-with-migration for issue types.
  - Local env files (http-client.env.json, http-client.private.env.json) are git-ignored by default.
  - Security and disposal of sensitive data:
    - Do not paste tokens directly into requests; always use environment variables (JIRA_API_TOKEN, JIRA_BASIC_AUTH).
    - Never commit environment files with real credentials. They are git-ignored, but verify they are not added to commits/PRs.
    - JetBrains HTTP Client keeps a request/response history under .idea/httpRequests. Clear the HTTP Client history and delete that folder after testing, especially on shared machines.
    - If a token may have been exposed (logs, screenshots, responses), rotate/revoke the JIRA_API_TOKEN immediately.
    - Avoid sharing saved responses that include Authorization headers or other secrets; redact before sharing.
    - Prefer http-client.private.env.json for local use and keep it outside any synced/shared directories.
