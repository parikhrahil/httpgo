# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-05-20

### Added

- `-t` short alias for `collection --timeout`.

### Fixed

- `collection --timeout <duration>` is now honored at runtime via a per-request
  `context.WithTimeout`, leaving the shared client's default 30s ceiling
  unchanged. Previously the flag was declared but ignored.
- Removed stale "not yet implemented" notes from `collection --dry-run` and
  `--timeout` help text.

## [1.0.0] - 2026-05-20

### Added

- `collection` (aliases: `cl`, `run`) subcommand to execute a named request
  from `~/.httpgo/collections/<namespace>/http`.
  - `-o, --output` to append the response body to a file.
  - `-p, --prettify` to pretty-print JSON response bodies (default `true`).
  - `-r, --raw` to print only the response body.
  - `-H, --include-headers` to include response headers in the output.
  - `-v, --vars KEY=VAL` / `-g, --global-vars KEY=VAL` to upsert variables
    into the namespace's env file or the shared `globalenv` before running.
  - `-u, --unset KEY` / `-U, --global-unset KEY` to delete variables before
    running. Clears are applied before overrides in the same invocation.
- `list` (alias: `ls`) subcommand to list namespaces, with `-a, --all` to
  also print every named request per namespace.
- `env` subcommand to print the shared `globalenv` or the merged variable
  set visible to a namespace.
- `wd` subcommand to print the collections working directory.
- `.http`-style request file parser supporting `###`-separated blocks,
  `# @name` / `// @name` request naming, and `{KEY}` interpolation from
  merged global + namespace env files (namespace wins on conflicts).
- Automatic `HTTP/1.1` version and `Content-Length` header injection for
  request blocks that omit them.
- Package-level `*http.Client` with connection pooling and a 30s overall
  timeout.
- `Makefile` with `build`, `install` (defaults to `~/.local/bin`), and
  `clean` targets.

[1.0.1]: https://github.com/parikhrahil/httpgo/releases/tag/v1.0.1
[1.0.0]: https://github.com/parikhrahil/httpgo/releases/tag/v1.0.0
