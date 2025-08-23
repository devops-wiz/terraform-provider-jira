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
