## Project
LightRDP is a remote help desk tool, similar to RustDesk and AnyDesk, built to be extremely lightweight and easy to use.

## Stack
- Go 1.26.9
- pion/webrtc v5
- coder/websocket v1

## Folder Structure
- `cmd/`: application entry points (`server` and `client`).
- `internal/`: private packages: peer connections, transport, WebSocket handling, signaling protocol, helpers.
- `docs/`: user-facing documentation (`app/` for usage, `errors/` for known errors).
- `specs/`: feature specs (`_TEMPLATE.md` is the template, `done/` holds merged specs).
- `.claude/commands/`: `/spec` creates a spec, `/implement` implements a spec and opens a PR.

## Conventions
- File names: use `snake_case`.
- Tests: every new feature must have tests.
- Tests: never write code that only exists to make tests pass (no special-casing tests, no weakened assertions).
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

## Do NOT
- Install dependencies without asking first.
- Open or expose ports on the machine to the network in an insecure way.
- Bypass or work around the code in any way. This code must be trustworthy.

## Commits and PRs
- Never add yourself as a co-author of commits or PRs, and never mention yourself in commit messages.
- In PR descriptions, state the AI tool and model used in the "AI usage" section of the template.
- Never commit directly to the `master` branch.
- When opening a new PR, use `.github/PULL_REQUEST_TEMPLATE.md`.
