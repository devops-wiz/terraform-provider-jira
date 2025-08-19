---
page_title: "jira Provider"
description: |-
  Jira provider for interacting with Jira instances using the go-jira library.
---

# jira Provider

Jira provider for interacting with Jira instances using the go-jira library.

## Usage

When configuring `jira` provider, there are a few ways to set the required attributes.

The provider block can be empty, and the required attributes can be set via environment variables:

```terraform
# Minimal provider configuration
# Credentials are sourced from environment variables:
#   JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN
# For self-hosted Jira (basic auth): JIRA_USERNAME, JIRA_PASSWORD

terraform {
  required_providers {
    jira = {
      source = "devops-wiz/jira"
      # version constraints can be added by end users
    }
  }
}

provider "jira" {}
```

with

```dotenv
# Replace placeholders with your actual values. Do NOT commit real tokens.
JIRA_ENDPOINT="https://<your-tenant>.atlassian.net"
JIRA_API_EMAIL="<your-email@example.com>"
JIRA_API_TOKEN="<your-api-token>"
```

The required attributes can also be directly set in the provider block

```terraform
provider "jira" {
  # Replace placeholders with your actual values. Do NOT commit real tokens.
  endpoint       = "https://<your-tenant>.atlassian.net"
  api_auth_email = "<your-email@example.com>"
  api_token      = "<your-api-token>"
}
```

There can also be a mixture of the two.

```terraform
provider "jira" {
  # Replace placeholders with your actual values. Do NOT commit real tokens.
  endpoint       = "https://<your-tenant>.atlassian.net"
  api_auth_email = "<your-email@example.com>"
}
```

with

```dotenv
# Replace placeholder with your actual API token. Do NOT commit real tokens.
JIRA_API_TOKEN="<your-api-token>"
```

For more information on which attributes can be set for the provider config, please see [Schema](#schema) below.

Additional guidance: see [Authentication methods](#authentication-methods) and [Troubleshooting](#troubleshooting) below.

## Authentication methods

API token (recommended for Jira Cloud):
- Attributes: endpoint, api_auth_email, api_token
- Environment variables: JIRA_ENDPOINT (alias: JIRA_BASE_URL), JIRA_API_EMAIL (alias: JIRA_EMAIL), JIRA_API_TOKEN
  - Precedence: provider attributes > canonical env vars > aliases

Example:
```terraform
provider "jira" {
  endpoint       = "https://example.atlassian.net"
  api_auth_email = "user@example.com"
  api_token      = "<token>"
}
```

Basic auth (primarily for self-hosted/server):
- Attributes: endpoint, username, password
- Environment variables: JIRA_ENDPOINT, JIRA_USERNAME, JIRA_PASSWORD

Example:
```terraform
provider "jira" {
  endpoint = "https://jira.example.internal"
  username = "admin"
  password = "<password>"
}
```

Notes:
- Only one auth method should be configured at a time (api_token or basic).
- Prefer API token for Jira Cloud for security and compatibility.

## HTTP status handling

- Success is any 2xx response.
- Create operations accept 200 OK or 201 Created as success.
- Update operations accept 200 OK or 204 No Content as success.
- Delete operations accept 200 OK or 204 No Content as success; 404 Not Found is treated as already-deleted (idempotent delete).
- Read operations that receive 404 Not Found will remove the resource from state without error.
- 429 Too Many Requests and 5xx responses are retried per provider retry settings and honor Retry-After when present.
- Diagnostics and debug logs are sanitized to avoid leaking secrets and query strings.

## Troubleshooting

- Authentication errors (401/403):
  - Verify endpoint and credentials are set via env vars or provider attributes.
  - For Cloud, prefer API token + email; for Server, use username/password.
  - Ensure the account has required permissions for the called APIs.
- Endpoint/base URL issues (404/connection errors):
  - Confirm the endpoint matches your Jira site (e.g., https://your-domain.atlassian.net).
  - Check organization/network proxies or VPNs if requests fail to connect.
- Rate limiting (429):
  - The provider retries 429/5xx by default and honors Retry-After when present. See [Rate limits and retries](#rate-limits-and-retries).
  - If you consistently hit 429, reduce concurrency (Terraform `-parallelism=1..5`), increase backoff or attempts, and consider staggering applies.
  - Typical mitigations: lower `retry_max_attempts` when the API enforces tight global quotas (to fail fast), or increase backoff windows when bursty traffic is acceptable.
- Timeouts / long operations:
  - Increase http_timeout_seconds or set operation_timeouts for specific CRUD phases.
  - Example: `operation_timeouts = { create = "2m", read = "30s" }`.
  - Remember: overall wall time is roughly `(retries + 1) × http_timeout_seconds + total_backoff`.
- Work type hierarchy levels:
  - On Standard editions, hierarchy_level supports only -1 (sub-task) or 0 (standard). Values >= 1 (e.g., 1 = Epic) require Jira Software Premium (Advanced Roadmaps).
  - See: https://support.atlassian.com/jira-software-cloud/docs/issue-type-hierarchy/ and https://support.atlassian.com/jira-software-cloud/docs/configure-issue-type-hierarchy/
  - If you are on Premium, manage hierarchy levels in Jira admin and omit hierarchy_level from Terraform; the provider will read the level from Jira.
- Import issues:
  - Ensure ID format matches the resource expectation (e.g., canonical IDs from Jira responses).
- Debugging:
  - Set TF_LOG=DEBUG when running Terraform. You may also set `debug=true` in the provider block for additional structured logs.
  - Use `go run . -debug` or `task delve` to run the provider in debug mode for deeper inspection.

### Sample debug logs (redacted)

- 429 with Retry-After:
  ```text
  [DEBUG] jira: request failed: HTTP 429 Too Many Requests; Headers: Retry-After=30; X-Request-Id=req-123
  [DEBUG] jira: retrying after server signal (Retry-After=30s) [attempt=2/5]
  ```
- 429 without Retry-After (backoff policy decides):
  ```text
  [DEBUG] jira: request failed: HTTP 429 Too Many Requests
  [DEBUG] jira: retrying with exponential backoff [attempt=3/5] backoff=2.5s
  ```
- Transient 5xx:
  ```text
  [DEBUG] jira: request failed: HTTP 503 Service Unavailable; X-Request-Id=req-999
  [DEBUG] jira: retrying with jittered backoff [attempt=2/4] backoff=1.2s
  ```
- Concurrency saturation (symptom):
  ```text
  [DEBUG] jira: multiple requests returning 429; consider lowering Terraform -parallelism and tuning provider retry settings
  ```

## Rate limits and retries

Atlassian Cloud may respond with HTTP 429 (rate limited) or transient 5xx errors. By default, this provider retries those responses safely:

- retry_on_429_5xx: enabled by default. When true, the provider uses a retrying HTTP client that retries on 429 and 5xx and honors the server's Retry-After header.
- retry_max_attempts: default 4 (maximum number of retries; total attempts = 1 initial + retries).
- retry_initial_backoff_ms: default 500ms.
- retry_max_backoff_ms: default 5000ms.

How retries/backoff work
- On HTTP 429 with Retry-After, the client waits for the server-specified delay (seconds) before retrying.
- On HTTP 429 without Retry-After, and on 5xx responses, the client uses capped exponential backoff with jitter between initial and max backoff.
- Retries stop on success, when attempts exceed retry_max_attempts, or for non-retryable errors (e.g., 4xx other than 429).

Tuning guidance
- To reduce total wall time under sustained rate limits: lower retry_max_attempts (fail fast) and/or lower Terraform `-parallelism`.
- To improve success under bursty load: increase retry_max_attempts moderately and widen backoff (initial/max) to spread retries.
- For CI stability, prefer conservative concurrency and wider backoff; for local iteration, consider fewer attempts to surface issues faster.

Concurrency and Terraform parallelism
- Terraform applies run multiple resources concurrently by default. Use `-parallelism=N` (e.g., 1–5) to reduce concurrent calls when hitting Jira org-wide rate limits.
- Some Jira tenants impose per-user or per-org quotas; coordinate with your team to avoid overlapping heavy runs.

Timeout interactions
- http_timeout_seconds applies per HTTP attempt. Overall wall clock ≈ `(retries + 1) × http_timeout_seconds + total_backoff`.
- For long CRUD operations, prefer per-operation timeouts via `operation_timeouts` and adjust retries to balance success vs speed.

Why 1–600 for http_timeout_seconds?
- 0 disables the Go net/http client timeout and risks hung plans. A minimum of 1 second avoids indefinite waits.
- 600 seconds (10 minutes) caps a single HTTP attempt to prevent runaway applies and aligns with common gateway/service timeouts. For long-running operations, prefer operation_timeouts for CRUD phases and tune retry settings. Overall wall time is approximately `(retries + 1) × http_timeout_seconds + total_backoff`.

Examples

Disable retries:
```terraform
provider "jira" {
  retry_on_429_5xx = false
}
```

Tune retry behavior:
```terraform
provider "jira" {
  retry_max_attempts       = 6
  retry_initial_backoff_ms = 750
  retry_max_backoff_ms     = 8000
}
```

Terraform CLI parallelism example:
```sh
terraform apply -parallelism=3
```

## Advanced Examples

Provider retries and timeouts:
```terraform
# Advanced provider configuration: retries and timeouts
# Credentials are sourced from environment variables:
#   JIRA_ENDPOINT, JIRA_API_EMAIL, JIRA_API_TOKEN
# For self-hosted Jira (basic auth): JIRA_USERNAME, JIRA_PASSWORD

provider "jira" {
  # Per-attempt HTTP timeout (seconds)
  http_timeout_seconds = 60

  # Retry policy for 429/5xx
  retry_on_429_5xx         = true
  retry_max_attempts       = 6
  retry_initial_backoff_ms = 750
  retry_max_backoff_ms     = 8000

  # Optional CRUD operation timeouts
  operation_timeouts = {
    create = "2m"
    read   = "30s"
    update = "2m"
    delete = "2m"
  }
}
```

Debug logging notes:
- Enable TF_LOG=DEBUG when running Terraform applies
- Or set provider-level debug=true to emit additional structured debug logs:

  Example provider block with debug flag:

  ```hcl
  provider "jira" {
    endpoint        = var.jira_endpoint
    api_auth_email  = var.jira_email
    api_token       = var.jira_api_token
    debug           = true
  }
  ```

- You can also run the provider with -debug (task delve or go run . -debug)
- Secrets are redacted by the provider; avoid printing tokens

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `api_auth_email` (String) Email address associated with the API token. **Required** when using API token authentication. Can be set with environment variable `JIRA_API_EMAIL` (canonical) or alias `JIRA_EMAIL`. Precedence: provider attributes > canonical env var > alias.
- `api_token` (String, Sensitive) API token (PAT) for authentication. **Required** when using API token authentication with email.Can be set with environment variable `JIRA_API_TOKEN`.
- `auth_method` (String) Authentication method to use for Jira. Default: "api_token". Accepts values `api_token` or `basic`.
- `debug` (Boolean) Enable additional provider debug logs. Honors TF_LOG for log level; when true, the provider emits extra structured debug logs with sensitive values redacted.
- `email_redaction_mode` (String) Controls how emails are sanitized in logs/errors. Default: "full". Allowed values: `full` (fully redact as "[REDACTED_EMAIL]") or `mask` (partially mask local-part and keep domain, e.g., "a****@example.com"). Can be set via environment variable `JIRA_EMAIL_REDACTION_MODE`. Precedence: provider attribute > env var.
- `endpoint` (String) Base Endpoint of the Jira client (e.g., 'https://your-domain.atlassian.net'). Can be set with environment variable `JIRA_ENDPOINT` (canonical) or alias `JIRA_BASE_URL`. Precedence: provider attributes > canonical env var > alias.
- `http_timeout_seconds` (Number) HTTP client timeout in seconds for all Jira API requests. Defaults to 30 seconds. Acceptable range is 1–600. Rationale: 0 disables the Go net/http client timeout and risks hung plans; a minimum of 1 second avoids indefinite waits. The 600-second (10 minute) maximum caps a single HTTP attempt to prevent runaway applies and aligns with typical upstream gateway/service limits. For long-running operations, prefer per-operation timeouts via operation_timeouts and consider retry/backoff settings—overall wall time includes (retries + 1) × http_timeout_seconds plus backoff.
- `operation_timeouts` (Attributes) Optional per-operation timeouts for provider-managed operations. Use Go duration strings like '30s', '2m', '1h'. Each value must be greater than 0 if set. (see [below for nested schema](#nestedatt--operation_timeouts))
- `password` (String, Sensitive) Password for basic authentication. **Required** when using basic authentication.Can be set with environment variable `JIRA_PASSWORD`.
- `retry_initial_backoff_ms` (Number) Initial backoff, in milliseconds, before the first retry. Defaults to 500 ms. Allowed range: 100–600000.
- `retry_max_attempts` (Number) Maximum number of retry attempts for transient failures. Defaults to 4. Allowed range: 1–10.
- `retry_max_backoff_ms` (Number) Maximum backoff, in milliseconds, for retries. Defaults to 5000 ms. Allowed range: 100–600000.
- `retry_on_429_5xx` (Boolean) Enable automatic retries on HTTP 429 and 5xx responses. Defaults to true.
- `username` (String) Username for basic authentication. **Required** when using basic authentication with password.Can be set with environment variable `JIRA_USERNAME`.

<a id="nestedatt--operation_timeouts"></a>
### Nested Schema for `operation_timeouts`

Optional:

- `create` (String) Timeout for create operations. Example: '2m'.
- `delete` (String) Timeout for delete operations. Example: '2m'.
- `read` (String) Timeout for read operations. Example: '30s'.
- `update` (String) Timeout for update operations. Example: '2m'.



