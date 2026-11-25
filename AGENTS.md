## Project
LightRDP is a remote help desk tool, similar to RustDesk and AnyDesk, built to be extremely lightweight and easy to use.

## Stack
- Go 1.26.9
- pion/webrtc v5
- coder/websocket v1

## Folder Structure
- `cmd/`: application entry point (`lightrdp`, with the `server`, `host`, `join` and `version` commands).
- `internal/`: private packages: CLI commands (`cli`), client runtime (`client`), signaling server, peer connections, transport, WebSocket handling, signaling protocol, helpers.
- `docs/`: user-facing documentation (`app/` for usage, `errors/` for known errors).
- `specs/`: feature specs (`_TEMPLATE.md` is the template, `done/` holds merged specs).
- `.claude/commands/`: `/spec` creates a spec, `/implement` implements a spec and opens a PR, `/security` runs the security review of a spec.
- `.claude/agents/`: `security_coordinator` and the reviewers it starts (`security_network`, `security_concurrency`, `security_errors`, `security_logic`).
- `.claude/skills/`: skills preloaded by the reviewers. The `hunt-*` and `triage-validation` skills come from Claude-BugHunter (see `.claude/THIRD_PARTY_NOTICES.md`) and are not edited here; `lightrdp-security-context` adapts them to LightRDP. Skill folders use kebab-case, as required by the skill format.

## Conventions
- File names: use `snake_case`.
- Tests: every new feature must have tests.
- Tests: never write code that only exists to make tests pass (no special-casing tests, no weakened assertions).
- Tests: with `coder/websocket`, a read whose context expires closes the connection. Never retry a read on the same connection after a timeout.
- Features: avoid changing other features when adding a new one. ALWAYS ask before making breaking changes to existing features.
- Changelog: always update `CHANGELOG.md` for code changes, with the date and version tag.
- README: keep `README.md` up to date with the current getting-started instructions, and link to the documentation pages (`docs/app/`, `docs/errors/`).

## Code Standards
- Follow the project's error convention, and document every new error in `docs/errors/`.
- Every new feature or contract change must update the usage documentation in `docs/app/`.
- Do not change contracts unnecessarily, and always follow the patterns already used in the codebase.
- Do not add comments to the code. Use clear names instead.

## Specs
- New features start as a spec in `specs/`, created from `specs/_TEMPLATE.md` (`/spec`).
- Only implement specs with `Status: Approved` (`/implement`).
- Spec status flow: `Draft` → `Approved` → `In review` → `Done`. Once merged, move the spec to `specs/done/`.
- Security review: `/spec` runs a design review and `/implement` runs a code review with `security_coordinator`, using the reviewers listed in the spec's "Security review" section. Findings introduced by a branch are fixed, with a test, before anything is committed or pushed. Findings in code the branch did not change are shown only to the author, never written in a spec, commit or pull request.

## Do NOT
- Install dependencies without asking first.
- Open or expose ports on the machine to the network in an insecure way.
- Bypass or work around the code in any way. This code must be trustworthy.

## Commits and PRs
- Never add yourself as a co-author of commits or PRs, and never mention yourself in commit messages.
- In PR descriptions, state the AI tool and model used in the "AI usage" section of the template.
- Never commit directly to the `master` branch.
- When opening a new PR, use `.github/PULL_REQUEST_TEMPLATE.md`.
- The branch name should not carry the spec number, only a description of what was done
