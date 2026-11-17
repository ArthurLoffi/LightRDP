# Changelog
All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased] - 2026-11-16
### Added
- Host access approval: the host is asked `Aceitar? [s/N]` for each connection request, and nothing is negotiated before it accepts. Requests are refused on any other answer (`recusado pelo host`) or after 60 seconds without an answer (`tempo esgotado aguardando o host`). Requests that arrive while the host is deciding get `host ocupado` (spec 0004).
- Signaling server (`cmd/server`) that registers clients by ID and relays offers, answers and ICE candidates over WebSocket.
- Client (`cmd/client`) with `-mode host|join`, `-id`, `-peer` and `-server` flags.
- Trickle ICE: candidates are sent as soon as they are found.
- Agent guidelines (`AGENTS.md`, `CLAUDE.md`), documentation structure (`docs/`) and pull request template.
- Documentation for known errors in `docs/errors/`.
- Usage guide (`docs/app/getting_started.md`) and chat page (`docs/app/chat/README.md`) (spec 0002).
- Tests for `internal/signaling`, `internal/config`, `internal/server`, `internal/transport`, `cmd/server` and `cmd/client`, and more tests for `internal/peer` (spec 0003).

### Changed
- `cmd/client`: the host no longer accepts connections automatically (spec 0004).
- `cmd/client`: stdin is read by a single line reader shared by the host confirmation and the chat (spec 0004).
- Project layout: `connection/` moved to `internal/peer`, `config/` moved to `internal/config`, client entry point moved to `cmd/client`.
- The client no longer requires copying and pasting the offer and answer by hand.
- README rewritten in English with the current usage.
- `cmd/server`: logger and HTTP server setup moved from `main()` to `newLogger` and `newServer` so they can be tested. Flags, defaults, messages and the `/ws` path are unchanged (spec 0003).
- Peer connection tests moved from `connection/` to `internal/peer` and updated to the new peer API (`New`, `OnConnectionStateChange`, `CreateOffer`, `HandleOffer`, `HandleAnswer`). Added a test for queued ICE candidates.

### Security
- The signaling server only relays `connect`, `offer`, `answer`, `candidate` and `reject` messages, so a client can no longer send fake server errors to another client.
- Control characters and escape sequences are removed from received chat messages before they are printed.
