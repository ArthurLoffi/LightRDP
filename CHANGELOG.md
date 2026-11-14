# Changelog
All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased] - 2026-11-13
### Added
- Signaling server (`cmd/server`) that registers clients by ID and relays offers, answers and ICE candidates over WebSocket.
- Client (`cmd/client`) with `-mode host|join`, `-id`, `-peer` and `-server` flags.
- Trickle ICE: candidates are sent as soon as they are found.
- Agent guidelines (`AGENTS.md`, `CLAUDE.md`), documentation structure (`docs/`) and pull request template.
- Documentation for known errors in `docs/errors/`.

### Changed
- Project layout: `connection/` moved to `internal/peer`, `config/` moved to `internal/config`, client entry point moved to `cmd/client`.
- The client no longer requires copying and pasting the offer and answer by hand.
- README rewritten in English with the current usage.
