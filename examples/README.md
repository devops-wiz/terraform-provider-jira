# Examples

This directory contains examples that are mostly used for documentation, but can also be run/tested manually via the Terraform CLI.

The documentation generator (tfplugindocs) looks for files in the following locations by default. All other .tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating runnable/testable examples even if some parts are not directly embedded in the docs.

- provider/provider.tf → example file for the provider index page (minimal)
- data-sources/`full data source name`/data-source.tf → example file for the named data source page (minimal)
- resources/`full resource name`/resource.tf → example file for the named resource page (minimal)

Per-scenario examples

- Provider scenarios (each folder contains a provider.tf):
  - examples/provider/api_token_cloud/
  - examples/provider/basic_auth/
  - examples/provider/mixed_env/
  - examples/provider/disable_retries/
  - examples/provider/tune_retries/
  - examples/provider/retry_and_timeouts/
  - examples/provider/explicit_attrs/

- Resource scenarios (per resource, per scenario folder containing resource.tf):
  - examples/resources/jira_work_type/update_description/resource.tf
  - examples/resources/jira_workflow_status/update_name/resource.tf
  - Import helpers:
    - examples/resources/jira_work_type/import.sh (shell script)
    - examples/resources/jira_work_type/import-by-identity.tf, import-by-string-id.tf
    - examples/resources/jira_workflow_status/import-by-identity.tf, import-by-string-id.tf, import.sh

- Data source scenarios (per scenario directories containing data-source.tf):
  - examples/data-sources/jira_work_types/filter_by_ids/data-source.tf
  - examples/data-sources/jira_work_types/filter_by_names/data-source.tf
  - examples/data-sources/jira_work_types/mixed_case_names/data-source.tf
