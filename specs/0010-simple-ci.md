# 0010: Simple CI

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-26 |
| Pull request | |

<!--
Status values:
- Draft: still being written, do not implement.
- Approved: ready to be implemented with /implement.
- In review: implemented, pull request open.
- Done: merged. Move the file to specs/done/.
-->

## Goal
Run the Go build, `go vet` and the test suite automatically on every pull request and on pushes to `master`, so a broken build or failing test is visible before a change is merged.

## Context
The repository has no CI. `.github/` only holds `PULL_REQUEST_TEMPLATE.md`. Every change is checked by hand, and spec 0007 shows the cost: a test that failed about 1 run in 3 went unnoticed until someone ran it 50 times. The project uses Go 1.26.9 (`go.mod`), and all tests are self-contained: TLS tests create their certificates in `t.TempDir()` and do not need the untracked `server.crt` or `server.key`, so CI needs no secrets or extra files.

## Scope
### In scope
- One GitHub Actions workflow in `.github/workflows/` that runs on pull requests and on pushes to `master`.
- Steps: check out the repository, install the Go version from `go.mod`, build (`go build ./...`), vet (`go vet ./...`), and test (`go test ./...`).
- A `CHANGELOG.md` entry under `Added`.

### Out of scope
- Race detector runs, coverage reports and coverage thresholds.
- Linters beyond `go vet` (for example `golangci-lint`), which would be a new tool dependency.
- Building or publishing release binaries, creating tags or GitHub releases.
- Branch protection rules and required status checks. These are GitHub repository settings, not files in the repository.
- Changes to application code, tests, or the signaling, transport or CLI behavior.
- Automatic merging, dependency update bots, and scheduled runs.

## Requirements
1. The workflow runs on `pull_request` and on `push` to `master`.
2. The workflow uses the Go version from `go.mod` (`go-version-file: go.mod`), so the CI toolchain never drifts from the project's declared version.
3. The workflow fails when `go build`, `go vet` or `go test` fails, and each step is visible on its own in the run log.
4. The workflow uses only read permissions for the repository token (`permissions: contents: read`).
5. The workflow does not use secrets and does not write to the repository.
6. The workflow file has no comments (`AGENTS.md`: no comments in code; use clear step names).

## User experience
Contributors see a check named after the workflow on each pull request, with a pass or fail result and a link to the failing step's log. There is no change for application users.

## Technical design
One new file, `.github/workflows/ci.yml` (name pending, see Open questions):

- `on:` `pull_request` and `push` with `branches: [master]`.
- `permissions:` `contents: read`.
- One job, `test`, on `ubuntu-latest`.
- Steps:
  1. `actions/checkout` (fetches the code).
  2. `actions/setup-go` with `go-version-file: go.mod`. Go module and build caching are enabled by the same action.
  3. `go build ./...`
  4. `go vet ./...`
  5. `go test ./...`

No change to `go.mod` or `go.sum`. No new Go dependency.

## Errors
None. CI produces no error messages that users of LightRDP see, so nothing is added to `docs/errors/`.

| Message | Cause | Solution |
|---|---|---|
| | | |

## Breaking changes
None.

## Dependencies
- GitHub-hosted runner `ubuntu-latest`.
- GitHub Actions `actions/checkout` and `actions/setup-go`, both maintained by GitHub. Exact versions are an open question. These are the only new external dependencies, and they run in CI only; nothing is installed on the developer's machine.

## Security review
Reviewers run by the security review (`.claude/commands/security.md`). Keep only the ones that apply: `network` (WebSocket and TLS, client and server), `concurrency` (race conditions), `errors` (exceptional conditions and error messages), `logic` (connection flow and business logic).

| Field | Value |
|---|---|
| Reviewers | network, concurrency, errors, logic |
| Design review | Pending |
| Code review | Passed (2026-11-26), 0 rechecks, 0 findings fixed |

## Acceptance criteria
- [x] `.github/workflows/ci.yml` contains one workflow that runs on `pull_request` and on `push` to `master`.
- [x] The workflow runs `go build ./...`, `go vet ./...` and `go test ./...` on Go from `go.mod` (`go-version-file: go.mod`). Run locally on the branch: `gofmt -l .` empty, `go vet ./...` clean, `go test -race ./...` passes.
- [ ] A pull request with a failing test shows a failing check, and the same pull request with the test fixed shows a passing check. Not checked yet: needs the pull request on GitHub.
- [x] The workflow has `permissions: contents: read` and uses no secrets.
- [x] The workflow file has no comments.
- [x] `CHANGELOG.md` has an `Added` entry under `[Unreleased]` with the date.
- [x] `go test ./...` still passes locally with no code change.

## Test plan
- Automated tests: the workflow itself is verified by GitHub runs: a pull request from a branch that is expected to pass, and a pull request with an intentionally failing test (closed without merging), to confirm the check turns red. No new Go tests, because no Go code changes.
- Manual tests: check the workflow file with a YAML linter or `actionlint` run locally, if the user approves installing it (see Open questions). Confirm the run log shows the five steps by name.

## Open questions
Resolved by the author on 2026-11-26:
- Workflow file: `ci.yml`.
- Platforms: `ubuntu-latest` only.
- Action versions: full commit SHAs. `actions/checkout` v7.0.1 (`3d3c42e5aac5ba805825da76410c181273ba90b1`) and `actions/setup-go` v7.1.0 (`d8cbb8045d7f43706b9177cceb36c1f91a6644d5`), the newest release tags at implementation time.
- Race detector in CI: no.
- Linting: `go vet` only, no `golangci-lint` or `staticcheck`.
- Triggers: pull requests and pushes to `master` only.
- Branch protection: the author sets `CI / test` as a required check on `master` after the first green run. Not in the repository.
- README badge: no.
- CHANGELOG entry: yes.
