# Changelog
All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Logo and README card in `docs/assets/`: the mark as SVG (`logo_mark.svg`), the horizontal logo for dark and light backgrounds (`logo_dark.png`, `logo_light.png`) and a 1280×640 card (`readme_card.png`) shown at the top of `README.md` (2026-11-23).

### Changed
- `README.md` is shorter: the title and intro already shown on the card, the full usage steps, the command list, the network rules and the stack list were removed or condensed into a quick start, and the details now live only in [Getting Started](docs/app/getting_started.md) and [Security](docs/app/security/README.md) (2026-11-23).

## [0.0.1] - 2026-11-21
Git tag: `v0.0.1`.

### Added
- `lightrdp` binary (`cmd/lightrdp`) built on `github.com/spf13/cobra`, with the `server`, `host`, `join HOST_ID`, `version` and `completion` commands. Every command has `--help` with a description, its flags and defaults, and examples. `lightrdp version` prints `lightrdp v0.0.1` (spec 0008).
- Usage errors (unknown command, unknown flag, wrong argument count) print `Error: ...` followed by the command usage and exit with code 2; runtime errors keep exit code 1 (spec 0008).
- `cmd/server`: `-cert` and `-key` flags serve the signaling server over TLS (`wss://`) on the same `/ws` path. The listen log line has a `tls` attribute (spec 0006).
- `cmd/client`: `wss://` support with certificate verification against the system roots plus the certificates given by the new `-ca` flag. Verification can never be skipped (spec 0006).
- Mandatory safety code: after the peer connection is `connected`, both clients show a 6-digit code (`NNN NNN`) derived from the two DTLS certificate fingerprints used in the handshake (host first, joiner second). The chat opens only after both users confirm it with `y` (spec 0006).
- `internal/peer`: `DTLSFingerprints` returns the local and remote DTLS certificate fingerprints of the handshake (spec 0006).
- Docs: `docs/app/security/` (creating a self-signed certificate with `openssl`, `wss://`, safety code) and new error pages for TLS, insecure URLs, `-ca` files and an unconfirmed safety code (spec 0006).
- Host access approval: the host is asked `Accept? [y/N]` for each connection request, and nothing is negotiated before it accepts. Requests are refused on any other answer (`refused by host`) or after 60 seconds without an answer (`timed out waiting for host`). Requests that arrive while the host is deciding get `host busy`. Answers typed before the prompt, or less than 1 second after it, are ignored so a late answer cannot accept a different requester (spec 0004).
- Signaling server (`cmd/server`) that registers clients by ID and relays offers, answers and ICE candidates over WebSocket.
- Client (`cmd/client`) with `-mode host|join`, `-id`, `-peer` and `-server` flags.
- Trickle ICE: candidates are sent as soon as they are found.
- Agent guidelines (`AGENTS.md`, `CLAUDE.md`), documentation structure (`docs/`) and pull request template.
- Documentation for known errors in `docs/errors/`.
- Usage guide (`docs/app/getting_started.md`) and chat page (`docs/app/chat/README.md`) (spec 0002).
- Tests for `internal/signaling`, `internal/config`, `internal/server`, `internal/transport`, `cmd/server` and `cmd/client`, and more tests for `internal/peer` (spec 0003).

### Changed
- **Breaking:** `go run ./cmd/server` and `go run ./cmd/client` were replaced by `go run ./cmd/lightrdp server|host|join`. `cmd/client` and `cmd/server` were removed (spec 0008).
- **Breaking:** `-mode host` is now `lightrdp host`, and `-mode join -peer HOST` is now `lightrdp join HOST`. `-mode` and `-peer` were removed (spec 0008).
- **Breaking:** long flags now take two dashes: `--bind`, `--port`, `--cert`, `--key`, `--debug`, `--server`, `--ca` and `--id`. Single-dash long flags are refused as unknown flags. Defaults are unchanged (spec 0008).
- **Breaking:** the client usage messages `usage: client -mode host|join ...` and `-peer is required with -mode join` were replaced by cobra usage errors. The TLS and CA messages use the new flag names: `--cert and --key must be used together`, `--cert and --key are required when --bind is not a loopback address` and `error: --ca: <reason>` (spec 0008).
- **Breaking (internal):** the runtime code moved from `package main` to importable packages: the server listen loop, logger and TLS checks to `internal/server` (`listen.go`), and the client session, chat, safety code and TLS loading to `internal/client`. The server listen loop returns an error instead of calling `os.Exit` (spec 0008).
- `github.com/spf13/cobra` v1.10.2 is a direct dependency (spec 0008).
- **Breaking:** `cmd/server` refuses to start when `-bind` is not a loopback address and `-cert`/`-key` are not set, or when only one of them is set (spec 0006).
- **Breaking:** `cmd/client` refuses `ws://` to any address that is not loopback; use `wss://` (spec 0006).
- **Breaking:** host approval merged with the safety code: the host is no longer asked before negotiation. It prints `Connection request from "<ID>"`, connects, and then asks `Accept "<ID>" with this code? [y/N]`; the joiner asks `Same code on the host? [y/N]`. A refusal on the joiner sends `reject` with `safety code not confirmed`. The 60-second timeout covers negotiation and the host answer. The requester now gets the host's ICE candidates and a DTLS connection before the host decides; no chat data passes before both confirm (spec 0006).
- `cmd/client`: the joiner accepts `reject` from the host until the chat opens, even after `connected`. On the host, a refusal from the joiner, a peer connection that fails before the chat opens, or an invalid offer, answer or candidate from the requester before the chat opens, ends that attempt and the host keeps waiting for new requests (spec 0006).
- `cmd/client`: both modes now read stdin from the start; the chat only forwards lines typed after it opens (spec 0006).
- **Breaking (internal):** `transport.Dial` takes a `*tls.Config` (spec 0006).
- **Breaking:** `cmd/server`: the `-addr` flag was replaced by `-bind` (default `127.0.0.1`) and `-port` (default `8080`). The server now only listens on the local machine by default; use `-bind <SERVER_IP>` or `-bind 0.0.0.0` to accept connections from the network.
- Every client message, server log line, flag description and error text is now in English, and the code comments were translated to English. Docs quote the new messages (spec 0005).
- **Breaking:** the host accepts a connection only with `y` or `Y` (`Accept? [y/N]`); `s` now refuses it. The texts inside `reject` and `error` payloads are now English; the protocol is unchanged, but scripts or log filters that match the old Portuguese text must be updated (spec 0005).
- `cmd/client`: the host no longer accepts connections automatically (spec 0004).
- `cmd/client`: stdin is read by a single line reader shared by the host confirmation and the chat (spec 0004).
- Project layout: `connection/` moved to `internal/peer`, `config/` moved to `internal/config`, client entry point moved to `cmd/client`.
- The client no longer requires copying and pasting the offer and answer by hand.
- README rewritten in English with the current usage.
- `cmd/server`: logger and HTTP server setup moved from `main()` to `newLogger` and `newServer` so they can be tested. Flags, defaults, messages and the `/ws` path are unchanged (spec 0003).
- Peer connection tests moved from `connection/` to `internal/peer` and updated to the new peer API (`New`, `OnConnectionStateChange`, `CreateOffer`, `HandleOffer`, `HandleAnswer`). Added a test for queued ICE candidates.

### Fixed
- `cmd/server`: `TestNewServerRoutesWebSocket` no longer fails intermittently. It retried a read on bob's connection after a 200 ms timeout, but `coder/websocket` closes the connection when a read context expires. The test now reads bob's connection once with the full test timeout and sends `connect` again only when the server answers alice with `target not found: bob` (spec 0007).

### Security
- Signaling traffic can be encrypted with TLS, and the client refuses unencrypted connections to other machines (spec 0006).
- The safety code lets users detect a DTLS fingerprint swapped in the SDP by a network attacker or a compromised signaling server (spec 0006).
- The signaling server only relays `connect`, `offer`, `answer`, `candidate` and `reject` messages, so a client can no longer send fake server errors to another client.
- Control characters and escape sequences are removed from received chat messages before they are printed.
