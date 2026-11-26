# 0008: Cobra CLI for client and server

| Field | Value |
|---|---|
| Status | Done |
| Author | ArthurLoffi |
| Created | 2026-11-20 |
| Pull request | https://github.com/ArthurLoffi/LightRDP/pull/13 |

<!--
Status values:
- Draft: still being written, do not implement.
- Approved: ready to be implemented with /implement.
- In review: implemented, pull request open.
- Done: merged. Move the file to specs/done/.
-->

## Goal
Replace the `flag`-based command lines of `cmd/server` and `cmd/client` with a single `lightrdp` binary built on `github.com/spf13/cobra`, with `server`, `host`, `join` and `version` commands, built-in help and clear usage errors.

## Context
Today each entry point parses flags by hand with the standard `flag` package:
- `cmd/server/main.go`: `-bind`, `-port`, `-debug`, `-cert`, `-key`, validated by `validateListenConfig`.
- `cmd/client/main.go`: `-mode host|join`, `-id`, `-peer`, `-server`, `-ca`, with manual checks that print `usage: client -mode host|join ...` or `-peer is required with -mode join` and exit with code 2.

The `-mode` flag mixes two different actions in one flag, and help output is only the raw `flag` defaults. A command tree makes the actions explicit and gives every command its own help and examples.

Related: Notion backlog card LRDP-14 "CLI do servidor e do cliente com Cobra".

Current state of the working tree: `github.com/spf13/cobra v1.10.2` (with `spf13/pflag` and `inconshreveable/mousetrap`) is already in `go.mod` as an indirect dependency, and an unfinished stub exists at `cli/cobra/cli.go` (does not compile). The implementation deletes the stub and the `cli/` folder; the CLI lives in `internal/cli/`.

## Scope
### In scope
- One binary, `lightrdp`, at `cmd/lightrdp/main.go`, with the commands `server`, `host`, `join` and `version`.
- Cobra command tree in `internal/cli/`.
- Moving the runtime code out of `package main` of `cmd/client` and `cmd/server` into importable packages (see Technical design), with its tests.
- Removing `cmd/client` and `cmd/server`.
- `--help` for every command, with a short description, a long description and at least one example.
- Same defaults as today: `--bind 127.0.0.1`, `--port 8080`, `--server ws://localhost:8080/ws`, generated 6-character ID when `--id` is empty.
- Same validation as today: TLS listen rules, `--ca` loading, host ID required for `join`.
- Documentation: `README.md`, `docs/app/getting_started.md`, `docs/app/security/README.md`, and the `docs/errors/` pages that quote the old commands or flags (`invalid_arguments.md`, `tls_required.md`, `tls_flags_incomplete.md`, `server_tls_files.md`, `insecure_server_url.md`, `untrusted_certificate.md`, `server_unreachable.md`, `peer_not_found.md`, `invalid_ca_file.md`).
- `AGENTS.md` folder structure: `cmd/` now has a single `lightrdp` entry point; `internal/` lists `cli`.
- `CHANGELOG.md`: release the current changes as `[0.0.1]` with the date, git tag `v0.0.1`.
- Notion documentation (after the PR is opened):
  - "Arquitetura do LightRDP": replace the "Flags do client" and "Flags do servidor" tables and the example commands with the new commands and `--` flags; the components diagram says "mesmo binário `lightrdp` para servidor, host e join".
  - Backlog card LRDP-14: link the spec and PR, and move the status along with the spec.

### Out of scope
- Any change to the signaling protocol, the peer connection, the safety code, host approval or the chat.
- Configuration files or environment variables (no `viper`).
- Compatibility aliases for the old single-dash flags or for `-mode`.
- Packaging, installers or release binaries.

## Requirements
1. `lightrdp server` accepts `--bind`, `--port`, `--debug`, `--cert` and `--key` with the same meaning and defaults as today's `-bind`, `-port`, `-debug`, `-cert`, `-key`.
2. `lightrdp host` accepts `--server`, `--ca` and `--id`, and behaves as today's `-mode host`.
3. `lightrdp join HOST_ID` takes the host ID as its only positional argument (replacing `-peer`), accepts `--server`, `--ca` and `--id`, and behaves as today's `-mode join`.
4. `join` with zero or more than one argument prints a usage error and exits with code 2.
5. `lightrdp version` prints `lightrdp v0.0.1` to stdout and exits with code 0.
5a. Cobra's default `completion` command stays enabled: `lightrdp completion bash|zsh|fish|powershell` prints the shell completion script.
6. Every command accepts `--help` / `-h` and prints its usage, flags with defaults, and at least one example. `lightrdp` with no command prints the root help.
7. Unknown commands, unknown flags and wrong argument counts print a usage error and exit with code 2. Runtime failures (TLS files, server unreachable, peer errors) keep exit code 1 and their current messages.
8. The listen safety rules stay exactly as today: no TLS on a non-loopback `--bind` is refused, and `--cert` without `--key` (or the opposite) is refused. Only the wording changes to the new flag names.
9. `--ca` behaves exactly as `-ca` today; certificate verification can never be skipped.
10. Runtime logic is moved, not rewritten. Only parsing and dispatch are new code.
11. No new flag that weakens security (no `--insecure`, no skip-verify).
12. Output after a command starts is unchanged (`Connected to server as "HOST"`, `Waiting for connections...`, `signaling server listening ... tls=true`, etc.).

## User experience
```
lightrdp server [--bind 127.0.0.1] [--port 8080] [--cert FILE --key FILE] [--debug]
lightrdp host [--server wss://IP:8080/ws] [--ca FILE] [--id ID]
lightrdp join HOST_ID [--server wss://IP:8080/ws] [--ca FILE] [--id ID]
lightrdp version
lightrdp completion bash|zsh|fish|powershell
lightrdp --help
lightrdp join --help
```

Example session:
```
go run ./cmd/lightrdp server --bind 192.168.0.10 --cert server.crt --key server.key
go run ./cmd/lightrdp host --id HOST --server wss://192.168.0.10:8080/ws --ca server.crt
go run ./cmd/lightrdp join HOST --server wss://192.168.0.10:8080/ws --ca server.crt
```

Usage error example:
```
$ lightrdp join
Error: accepts 1 arg(s), received 0
Usage:
  lightrdp join HOST_ID [flags]
...
```

## Technical design
- `cmd/lightrdp/main.go`: calls `cli.Execute`, maps the returned error to the exit code (0, 1 or 2) and prints it.
- `internal/cli/` (files in `snake_case`): `root_command.go`, `server_command.go`, `host_command.go`, `join_command.go`, `version_command.go`, `exit_code.go`, with `_test.go` files.
  - Each command is built by a constructor function with its own flags (no package-level command globals), so each test builds a fresh tree.
  - Commands use `RunE`. The root sets `SilenceUsage` and `SilenceErrors`; usage errors (cobra argument and flag errors) are wrapped so `main` exits with 2 and prints the usage, runtime errors exit with 1.
  - The command constructors take the runtime functions as parameters so tests check parsing without opening ports or dialing.
  - The version string is a constant in `internal/cli`: `v0.0.1`.
- Runtime code moved out of `package main`, behavior and tests unchanged:
  - Server: `newLogger`, `newServer`, `serve`, `validateListenConfig` and the listen/shutdown loop from `cmd/server/main.go` move to `internal/server` (new file `listen.go`), exported as needed. `cmd/server/main_test.go` and `tls_test.go` move with them.
  - Client: `run`, `loadTLSConfig`, `session.go`, `chat.go`, `safety_code.go` and their tests move from `cmd/client` to a new package `internal/client`.
- `go.mod`: `github.com/spf13/cobra` moves from indirect to a direct requirement. `go mod tidy`.
- `cli/cobra/cli.go` is deleted.

## Errors
Each must be documented in `docs/errors/`.

| Message | Cause | Solution |
|---|---|---|
| `Error: accepts 1 arg(s), received 0` (or `received 2`, etc.) | `lightrdp join` without the host ID, or with more than one argument. | Run `lightrdp join <HOST_ID>`. Update `invalid_arguments.md`. |
| `Error: unknown command "<name>" for "lightrdp"` | Command typed wrong. | Run `lightrdp --help`. Add to `invalid_arguments.md`. |
| `Error: unknown flag: --<name>` / `Error: unknown shorthand flag: '<c>' in -<flag>` | Flag typed wrong, or old single-dash flag such as `-bind` or `-mode`. | Run `lightrdp <command> --help`; use `--` flags. Add to `invalid_arguments.md`. |
| `--cert and --key must be used together` | Replaces `-cert and -key must be used together`. | Update `tls_flags_incomplete.md`. |
| `--cert and --key are required when --bind is not a loopback address` | Replaces `-cert and -key are required when -bind is not a loopback address`. | Update `tls_required.md`. |
| `error: --ca: <reason>` | Replaces `error: -ca: <reason>`. | Update `invalid_ca_file.md`. |
| Removed: `usage: client -mode host|join ...` and `-peer is required with -mode join` | Replaced by the cobra usage errors above. | Remove from `invalid_arguments.md`. |

## Breaking changes
Approved by the author on 2026-11-20.
- **Binary:** `go run ./cmd/server` and `go run ./cmd/client` are replaced by `go run ./cmd/lightrdp server|host|join`. `cmd/client` and `cmd/server` are removed.
- **Commands:** `-mode host` becomes `lightrdp host`; `-mode join -peer HOST` becomes `lightrdp join HOST`. `-mode` and `-peer` are removed.
- **Flag syntax:** long flags now take two dashes (`--bind`, `--port`, `--cert`, `--key`, `--debug`, `--server`, `--ca`, `--id`). Single-dash long flags stop working.
- **Error messages:** the two client usage messages are replaced by cobra's; the TLS and `--ca` messages use the new flag names.
- **Internal:** runtime code moves from `cmd/client` and `cmd/server` to `internal/client` and `internal/server`.

## Dependencies
- `github.com/spf13/cobra v1.10.2` (already in `go.mod` as indirect; becomes direct). Requested by the author.
- Transitive: `github.com/spf13/pflag v1.1.0`, `github.com/inconshreveable/mousetrap v1.1.0` (Windows only).

## Acceptance criteria
- [x] `lightrdp server`, `lightrdp host` and `lightrdp join HOST_ID` behave as today's `cmd/server`, `cmd/client -mode host` and `cmd/client -mode join -peer HOST_ID`.
- [x] `lightrdp version` prints `lightrdp v0.0.1`.
- [x] Defaults are unchanged (`127.0.0.1`, `8080`, `ws://localhost:8080/ws`, generated ID).
- [x] TLS listen rules and `--ca` behavior are unchanged; no way to skip verification exists.
- [x] `--help` works on every command and shows an example.
- [x] Usage errors exit with code 2; runtime errors exit with code 1.
- [x] Moved tests pass in their new packages; new tests cover `internal/cli`.
- [x] `README.md`, `AGENTS.md`, `docs/app/` and `docs/errors/` use only the new commands and flags.
- [x] `CHANGELOG.md` has a `[0.0.1]` section with the date, breaking changes marked.
- [x] Notion "Arquitetura do LightRDP" and card LRDP-14 updated.
- [x] `cmd/client`, `cmd/server` and `cli/` removed; `go build ./...`, `go vet ./...` and `go test ./...` pass.

## Test plan
- Automated tests:
  - Build a fresh command tree per test, set args with `SetArgs`, capture output with `SetOut`/`SetErr`, and inject fake runtime functions.
  - `server`: defaults reach the runtime function; `--cert` without `--key` and non-loopback `--bind` without TLS return the new messages; no real port is opened on the network.
  - `host`/`join`: defaults reach the runtime function; `join HOST` passes `HOST`; `join` with 0 or 2 arguments returns a usage error; `--ca` with an invalid file returns the `--ca` error.
  - `version` prints `lightrdp v0.0.1`.
  - `completion bash` prints a non-empty script.
  - Unknown command, unknown flag and old `-mode` flag return usage errors; exit code mapping (0, 1, 2) tested in `exit_code.go`.
  - `--help` output of each command contains its flags and example.
  - Moved tests from `cmd/client` and `cmd/server` keep passing.
- Manual tests:
  - Run `lightrdp server`, `lightrdp host` and `lightrdp join` on loopback and complete a chat with the safety code.
  - Run `lightrdp server --bind <LAN_IP> --cert server.crt --key server.key` and connect with `wss://` and `--ca`.
  - Check `lightrdp`, `lightrdp --help`, `lightrdp join --help`, `lightrdp version`, and a typo command.

## Open questions
- None. Decided on 2026-11-21: keep cobra's `completion` command; runtime code moves to `internal/server/listen.go` and a new `internal/client` package.
