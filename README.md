# httpgo

A scriptable HTTP client driven by `.http`-style files on disk. Write your
requests once, group them into namespaces, and run them by name from your
terminal. Variables live in plain `KEY=value` files, and per-namespace values
transparently override shared globals.

```bash
httpgo collection users get-profile
httpgo run users get-profile -v userId=42 -g host=https://api.example.com
```

## Features

- Plain-text `.http` request files — diff-friendly, version-controllable, no GUI required.
- Per-namespace and shared global variables with `{KEY}` interpolation.
- Upsert or delete env values inline (`--vars`, `--global-vars`, `--unset`, `--global-unset`).
- JSON pretty-printing, raw body mode, optional response-header dump.
- Append the response body to a file with `--output`.
- Cobra-based CLI with short aliases (`cl`, `run`, `ls`).

## Installation

### Prerequisites

- Go 1.26 or newer (see `go.mod`).

### Build from source

```bash
git clone <repo-url> httpgo
cd httpgo
go build -o httpgo .
```

This produces an `httpgo` binary in the repository root. Copy it anywhere on
your `PATH`.

### Install via Makefile

The bundled `Makefile` builds and installs the binary into `~/.local/bin` by
default:

```bash
make install
```

To install elsewhere, override `INSTALL_DIR`:

```bash
make install INSTALL_DIR=/usr/local/bin
```

If the install directory is not yet on your `PATH`, the target will print the
line to add to your shell profile.

### Run without installing

```bash
go run . list
go run . collection <namespace> <request>
```

## Quick start

```bash
# 1. Initialize the working directory by running any command. The first
#    invocation creates ~/.httpgo/collections and an empty globalenv file.
httpgo wd
# /Users/you/.httpgo/collections

# 2. Create your first namespace.
mkdir -p ~/.httpgo/collections/demo

# 3. Drop a request into ~/.httpgo/collections/demo/http
cat > ~/.httpgo/collections/demo/http <<'EOF'
### get-ip
# @name get-ip
GET {host}/ip
Accept: application/json
EOF

# 4. Set the shared host variable in the global env file.
echo 'host=https://httpbin.org' >> ~/.httpgo/collections/globalenv

# 5. Run it.
httpgo collection demo get-ip
```

## Collection layout

Every collection lives under a single fixed working directory:

```
~/.httpgo/collections/
  globalenv               # KEY=value pairs shared across all namespaces
  <namespace>/
    http                  # one or more "###"-separated named request blocks
    env                   # KEY=value pairs scoped to this namespace
  <another-namespace>/
    http
    env
```

- The working directory is **fixed** at `~/.httpgo/collections`. There is no
  flag or environment variable to relocate it.
- A *namespace* is any immediate subdirectory; its name is what you pass to
  the `collection`, `list`, and `env` subcommands.
- The two files inside a namespace are literally named `http` (no extension)
  and `env` (no extension).
- Print the resolved path any time with `httpgo wd`.

## Authoring requests

Each namespace's `http` file is a flat list of request blocks separated by
lines that start with `###`. A block looks like this:

```http
### Anything after the hashes is ignored — it's just a visual separator
# @name list-users
GET {host}/users?page={page}
Accept: application/json
Authorization: Bearer {token}

###
# @name create-user
POST {host}/users
Content-Type: application/json

{
  "name": "Ada",
  "email": "ada@example.com"
}
```

Rules to remember:

- **Names come from `@name` comments only.** Either `# @name foo` or
  `// @name foo` works. A bare `### foo` line is *not* a name — it's just a
  separator.
- **Interpolation uses single braces:** `{KEY}` is replaced by the matching
  value from the namespace's `env` file (or `globalenv` if the namespace
  doesn't define it). Substitution is a literal string replace, so avoid
  unescaped braces in payloads that aren't variables.
- **HTTP version is optional** on the request line. If you leave it off,
  `HTTP/1.1` is appended automatically.
- **`Content-Length` is auto-injected** if your block has a body but no
  explicit length header — without it, request bodies would silently be sent
  as zero bytes.
- The file may end with or without a trailing `###` separator; both are fine.

## Variables

Variables are plain `KEY=value` lines in two scopes:

- **Global** — `~/.httpgo/collections/globalenv`
- **Namespace** — `~/.httpgo/collections/<namespace>/env`

When a request is executed, both files are merged. Namespace values win on
key conflicts. Inline `# comments` and `"quoted values"` are supported.

```ini
# globalenv
host=https://api.example.com
timeout=30   # inline comment, stripped at load time
```

```ini
# <namespace>/env
host=https://staging.api.example.com   # overrides global for this namespace
token=abc123
```

### Inspecting variables

```bash
httpgo env              # print globalenv
httpgo env <namespace>  # print the merged set the namespace sees
```

### Mutating variables from the CLI

You can upsert or delete variables as part of any `collection` run. Clears
are applied **before** overrides, so the same key can be cleared and re-set
in one invocation.

```bash
# Set / upsert
httpgo run demo get-ip -v token=xyz -v userId=42
httpgo run demo get-ip -g host=https://api.example.com

# Delete
httpgo run demo get-ip -u token
httpgo run demo get-ip -U host

# Both at once: clears first, then sets — useful for rotating a value.
httpgo run demo get-ip -u token -v token=new-value
```

The CLI flags **persist** to the underlying env file — there are no
ephemeral, run-only overrides. If you want a one-off value, run with the
override and then `--unset` it afterwards.

## Commands

### `collection <namespace> <request>`

Aliases: `cl`, `run`. Executes a named request.

| Flag | Description |
| --- | --- |
| `-o, --output <path>` | Append the response body to a file. |
| `-p, --prettify` | Pretty-print JSON response bodies. Default: `true`. |
| `-r, --raw` | Print only the response body. Suppresses the request dump and status line. |
| `-H, --include-headers` | Include response headers in the printed output. |
| `-v, --vars KEY=VAL` | Upsert into the namespace's `env` file. Repeatable. |
| `-g, --global-vars KEY=VAL` | Upsert into the shared `globalenv`. Repeatable. |
| `-u, --unset KEY` | Delete a key from the namespace's `env` file. Repeatable. |
| `-U, --global-unset KEY` | Delete a key from the shared `globalenv`. Repeatable. |
| `--dry-run` | Declared but not yet honored. |
| `--timeout <duration>` | Declared but not yet honored. |

### `list [namespace]`

Alias: `ls`. Lists namespaces.

```bash
httpgo list                # just the namespace names
httpgo ls --all            # every namespace, with its request names
httpgo ls <namespace>      # one namespace, with its request names
```

### `env [namespace]`

Prints variables.

```bash
httpgo env                 # globalenv only
httpgo env <namespace>     # globalenv merged with <namespace>/env (namespace wins)
```

### `wd`

Prints the absolute path of the collections working directory.

```bash
httpgo wd
# /Users/you/.httpgo/collections
```

## Examples

### A typical workflow

```bash
# See what's available.
httpgo list --all

# Run a request.
httpgo run users get-profile

# Run with overrides — both values persist to the env files.
httpgo run users get-profile -v userId=99 -g host=https://staging.example.com

# Just the body, please — useful for piping into jq or saving to disk.
httpgo run users get-profile --raw | jq '.id'
httpgo run users get-profile --output /tmp/profile.json
```

### A POST with a JSON body

```http
### Create a user
# @name create-user
POST {host}/users
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "Grace",
  "role": "admin"
}
```

```bash
httpgo run users create-user
```

### A form-encoded POST

```http
### Login
# @name login
POST {host}/login
Content-Type: application/x-www-form-urlencoded

username={username}&password={password}
```

```bash
httpgo run auth login -v username=alice -v password=s3cret
```

Form-encoded request bodies are printed as a parsed key/value list rather
than the raw payload, which keeps secrets a touch less conspicuous in logs.

## Troubleshooting

- **`no collection found for <name>`** — the namespace directory does not
  exist under `~/.httpgo/collections`. Run `httpgo list` to see what is
  actually available.
- **`request with name "<name>" not found`** — the `http` file has no block
  carrying a matching `# @name <name>` comment. Remember: the `###`
  separator does **not** set the name.
- **POST/PUT body arrives empty on the server** — make sure the block has a
  blank line between the headers and the body. `Content-Length` is injected
  for you, but only if the parser can find the header/body separator.
- **Unexpected `{` substitution** — `{KEY}` interpolation is a literal
  string replace. If your payload legitimately contains `{KEY}` text that
  isn't meant to be a variable, rename the variable or use a different
  marker syntax in your data.

## License

See [LICENSE](LICENSE).
