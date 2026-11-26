# Changelog
All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Add the errors `safety code commitment mismatch` and `safety code handshake failed: <detail>`. The joiner exits with `safety code handshake failed: timed out` when the safety code exchange does not finish 60 seconds after `connected` (spec 0009).

### Changed
- **BREAKING:** Derive the safety code from the two DTLS fingerprints and a random nonce from each client, and show it after the nonce exchange instead of right after `connected`. Clients from before this change cannot connect to clients with it (spec 0009).
- Ignore `lightrdp:safety-code-confirmed` until the nonce exchange is done (spec 0009).

### Security
- Prevent a man in the middle from matching the safety code by searching for DTLS certificates until the codes on the two computers collide. Before the code is shown, the clients exchange random nonces with `lightrdp:commit:`, `lightrdp:nonce:` and `lightrdp:reveal:`, and the host checks the revealed nonce against the commitment (spec 0009).

## [0.0.1] - 2026-11-21
Git tag: `v0.0.1`.

### Added
- Add the `lightrdp` binary with the `server`, `host`, `join HOST_ID`, `version` and `completion` commands. Every command has `--help` with a description, its flags and defaults, and examples, and `lightrdp version` prints `lightrdp v0.0.1` (spec 0008).
- Print `Error: ...` followed by the command usage and exit with code 2 on usage errors (unknown command, unknown flag, wrong argument count). Runtime errors keep exit code 1 (spec 0008).
- Add `-cert` and `-key` to `cmd/server` to serve the signaling server over TLS (`wss://`) on the same `/ws` path. The listen log line has a `tls` attribute (spec 0006).
- Add `wss://` support to `cmd/client`, with certificate verification against the system roots plus the certificates given by the new `-ca` flag. Verification can never be skipped (spec 0006).
- Add a mandatory safety code: after the peer connection is `connected`, both clients show a 6-digit code (`NNN NNN`) derived from the two DTLS certificate fingerprints, and the chat opens only after both users confirm it with `y` (spec 0006).
- Add host access approval: the host is asked `Accept? [y/N]` for each connection request, and nothing is negotiated before it accepts. Requests are refused on any other answer (`refused by host`) or after 60 seconds without an answer (`timed out waiting for host`), and requests that arrive while the host is deciding get `host busy`. Answers typed before the prompt, or less than 1 second after it, are ignored so a late answer cannot accept a different requester (spec 0004).
- Add the signaling server (`cmd/server`), which registers clients by ID and relays offers, answers and ICE candidates over WebSocket.
- Add the client (`cmd/client`) with the `-mode host|join`, `-id`, `-peer` and `-server` flags.
- Send ICE candidates as soon as they are found (trickle ICE).

### Changed
- **BREAKING:** Replace `go run ./cmd/server` and `go run ./cmd/client` with `go run ./cmd/lightrdp server|host|join`, and remove `cmd/client` and `cmd/server` (spec 0008).
- **BREAKING:** Replace `-mode host` with `lightrdp host` and `-mode join -peer HOST` with `lightrdp join HOST`, and remove `-mode` and `-peer` (spec 0008).
- **BREAKING:** Require two dashes for long flags: `--bind`, `--port`, `--cert`, `--key`, `--debug`, `--server`, `--ca` and `--id`. Single-dash long flags are refused as unknown flags, and defaults are unchanged (spec 0008).
- **BREAKING:** Replace the client usage messages `usage: client -mode host|join ...` and `-peer is required with -mode join` with cobra usage errors, and use the new flag names in the TLS and CA messages: `--cert and --key must be used together`, `--cert and --key are required when --bind is not a loopback address` and `error: --ca: <reason>` (spec 0008).
- **BREAKING:** Refuse to start `cmd/server` when `-bind` is not a loopback address and `-cert`/`-key` are not set, or when only one of them is set (spec 0006).
- **BREAKING:** Refuse `ws://` in `cmd/client` to any address that is not loopback; use `wss://` (spec 0006).
- **BREAKING:** Merge host approval with the safety code: the host prints `Connection request from "<ID>"`, connects, and then asks `Accept "<ID>" with this code? [y/N]`, and the joiner asks `Same code on the host? [y/N]`. A refusal on the joiner sends `reject` with `safety code not confirmed`, and the 60-second timeout covers negotiation and the host answer. The requester now gets the host's ICE candidates and a DTLS connection before the host decides, but no chat data passes before both users confirm (spec 0006).
- Let the joiner accept `reject` from the host until the chat opens, even after `connected`. On the host, a refusal from the joiner, a peer connection that fails before the chat opens, or an invalid offer, answer or candidate from the requester before the chat opens ends that attempt, and the host keeps waiting for new requests (spec 0006).
- Forward only the chat lines typed after the chat opens (spec 0006).
- **BREAKING:** Replace the `-addr` flag of `cmd/server` with `-bind` (default `127.0.0.1`) and `-port` (default `8080`). The server now listens only on the local machine by default; use `-bind <SERVER_IP>` or `-bind 0.0.0.0` to accept connections from the network.
- Translate every client message, server log line, flag description and error text to English (spec 0005).
- **BREAKING:** Accept a connection on the host only with `y` or `Y` (`Accept? [y/N]`); `s` now refuses it. The texts inside `reject` and `error` payloads are now in English, so scripts or log filters that match the old Portuguese text must be updated (spec 0005).
- Stop accepting connections automatically on the host (spec 0004).
- Remove the need to copy and paste the offer and answer by hand.

### Security
- Allow encrypting signaling traffic with TLS, and refuse unencrypted connections to other machines (spec 0006).
- Let users detect, with the safety code, a DTLS fingerprint swapped in the SDP by a network attacker or a compromised signaling server (spec 0006).
- Relay only `connect`, `offer`, `answer`, `candidate` and `reject` messages on the signaling server, so a client can no longer send fake server errors to another client.
- Remove control characters and escape sequences from received chat messages before they are printed.
