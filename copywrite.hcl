project {
  license = "MPL-2.0"
  spdx = true
  copyright_holder = "DevOps Wiz"

  # Only operate on Go source by default; other content is excluded.
  header_ignore = [
    ".git/**",
    ".github/**",
    ".junie/**",
    ".idea/**",
    "vendor/**",
    "docs/**",
    "docs-internal/**",
    "templates/**",
    "examples/**",
    "scratch/**",
    "**/*.md",
    "**/*.tf",
    "**/*.http",
    "**/*.yml",
    "**/*.yaml",
    "**/*.json",
    "**/*.sh",
    "go.mod",
    "go.sum",
    "Taskfile.yml",
  ]
}
