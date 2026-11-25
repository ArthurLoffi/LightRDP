# 0009: Safety code commitment with nonces

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-24 |
| Pull request | |

<!--
Status values:
- Draft: still being written, do not implement.
- Approved: ready to be implemented with /implement.
- In review: implemented, pull request open.
- Done: merged. Move the file to specs/done/.
-->

## Goal
Stop a man in the middle from searching for DTLS certificates until the safety codes on the two computers match. Before the code is shown, both clients exchange random nonces over the DataChannel with a commit, nonce, reveal protocol, and the code is derived from the two fingerprints and the two nonces.

## Context
Spec 0006 added the safety code: `safetyCode(hostFingerprint, joinerFingerprint)` in `internal/client/safety_code.go` takes `sha256(hostFingerprint + joinerFingerprint)`, keeps the first 8 bytes and reduces them `% 1_000_000` (about 20 bits). The code is shown as soon as the peer connection is `connected` (`session.onPeerState` → `session.askSafetyCode` in `internal/client/session.go`).

Neither side commits to anything before it sees the other side's value. A man in the middle (on the network or a compromised signaling server) runs two DTLS sessions, one with each user. It learns the real fingerprints of both users from the offer and the answer, and only then picks the certificate it shows to one side. It can generate certificates until `safetyCode(hostFP, fakeJoinerFP) == safetyCode(fakeHostFP, joinerFP)`. That takes about 1 million attempts on average (one signature and one SHA-256 each), which fits in seconds on one machine and inside the 60 second `hostConfirmTimeout`. If it finds a match, both users see the same code, confirm it, and the attack is not detected.

With a commitment, the code also depends on a random nonce from each side. The side that reveals last commits to its nonce (`SHA-256(nonce)`) before it sees the other nonce. The fingerprints are fixed by the DTLS handshake before any nonce exists. The attacker can no longer search for a match: for each connection attempt, the codes on the two screens match with a probability of 1 in 1,000,000, and every failed attempt is visible to the users. This is the same idea as ZRTP and Bluetooth "Numeric Comparison", which also shows 6 digits.

Related: spec 0006 (signaling encryption and safety code), Notion card LRDP-15. Found by the automated security review of spec 0008 (PR #13).

## Scope
### In scope
- Commit, nonce, reveal exchange over the existing `chat` DataChannel, after it opens and before the safety code is shown.
- Code derived from the two DTLS fingerprints and the two nonces, in a fixed order on both sides.
- Refusing the attempt when the revealed nonce does not match the commitment, or when a handshake message is malformed or out of order.
- Tests for the new functions, the exchange, and the full host and join flow.
- `docs/app/security/README.md`, a new page in `docs/errors/`, `README.md` (safety code line) and `CHANGELOG.md`.

### Out of scope
- Changing WebRTC/DTLS, the signaling protocol or the signaling server.
- Changing the code length or format (stays `NNN NNN`).
- Host access password (separate card).
- Compatibility with clients from before this change.

## Requirements
1. After the `chat` DataChannel opens, the joiner generates a 32-byte nonce `nA` with `crypto/rand` and sends `lightrdp:commit:<hex SHA-256(nA)>`.
2. The host accepts exactly one commit. After it receives the commit, it generates a 32-byte nonce `nB` with `crypto/rand` and sends `lightrdp:nonce:<hex nB>`. The host never sends `nB` before it has the commit.
3. After the joiner receives `nB`, it sends `lightrdp:reveal:<hex nA>`. The joiner now has both nonces.
4. The host checks that `SHA-256(nA)` equals the commitment, using a constant-time comparison. If it matches, the host has both nonces.
5. Each side derives the code with `SHA-256(hostFingerprint || joinerFingerprint || nA || nB)`, first 8 bytes, `% 1_000_000`, shown as `NNN NNN`. The fingerprints still come from the DTLS handshake (`peer.DTLSFingerprints`), not from the SDP.
6. The safety code and the `[y/N]` question are shown only after the exchange completes. Reaching `connected` alone no longer shows the code.
7. A handshake message that is out of order, repeated, not valid hex, or not 32 bytes after decoding is a protocol error. A reveal that does not match the commitment is a commitment error. Both end the attempt the same way an invalid remote message does today: the joiner exits with the error, the host prints it and keeps waiting for new requests.
8. `lightrdp:safety-code-confirmed` is only accepted after the exchange completes. Chat messages are still only shown after both users confirm.
9. New nonces are generated for every connection attempt and are never reused.

## User experience
Nothing changes when the connection is honest: both computers show `Safety code: NNN NNN` and the same questions as today. The code appears a moment later, after the DataChannel opens instead of when the state becomes `connected`.

When the commitment check fails, the attempt ends with an error:
- Joiner: `error: safety code commitment mismatch`
- Host: `Request from "<ID>" dropped: safety code commitment mismatch`, then `Waiting for connections...`

## Technical design
- `internal/client/safety_code.go`: `safetyCode(hostFingerprint, joinerFingerprint string, joinerNonce, hostNonce []byte) string`. New `newNonce() ([]byte, error)`, `commitNonce(nonce []byte) [32]byte` and `verifyCommitment(commitment, nonce []byte) bool` (`crypto/subtle`).
- New `internal/client/safety_handshake.go`: the `lightrdp:commit:`, `lightrdp:nonce:` and `lightrdp:reveal:` prefixes, the per-attempt exchange state (role, local nonce, remote commitment, received nonce, done), and the parsing and validation of each message. Pure logic, no I/O, so it can be unit tested.
- `internal/client/chat.go`:
  - `chat` gets the exchange state and a `report func(safetyHandshakeResult)` set by the session.
  - `onChannelOpen`: the joiner starts the exchange.
  - `onMessage`: before the confirmation check, handshake messages go to the exchange. When it ends, the chat reports `safetyHandshakeResult{chat, joinerNonce, hostNonce, err}` to the session, outside the chat lock. `safetyCodeConfirmedMessage` is ignored until the exchange is done, and every message is ignored after the exchange failed.
- `internal/client/session.go`:
  - `session` gets a `handshakes chan safetyHandshakeResult`, created in `newSession`, and every new `chat` in `startPeer` reports to it.
  - `onPeerState` on `connected` only stores `connected`, and on the joiner starts the 60 second `safetyHandshakeTimeout`.
  - `loop` gets a new case for `s.handshakes` and one for the joiner handshake timeout. It ignores events from a dropped attempt (`ev.chat != s.chat`, like `ev.peer != s.peer`), sends errors to the same path as invalid remote messages, and calls `askSafetyCode(joinerNonce, hostNonce)` on success.
  - `askSafetyCode` passes the nonces to `safetyCode`.
- No change in `internal/signaling`, `internal/peer` or the server: everything goes over the DataChannel, which is already encrypted by DTLS.

## Errors
| Message | Cause | Solution |
|---|---|---|
| `safety code commitment mismatch` | The nonce revealed by the client does not match the commitment it sent first. Someone between the two computers changed the messages, or the other client is broken. | Do not continue. Check the server address and certificate, and try again. If it happens again, do not use that network or that server. |
| `safety code handshake failed: <detail>` | A handshake message was malformed, repeated or out of order. Usually the other side runs an older or different version of LightRDP. | Make sure both computers run the same LightRDP version, then try again. |

Both go in `docs/errors/` with links in `docs/errors/README.md`.

## Breaking changes
- Clients from before this change never send or answer the handshake messages, so they cannot connect to clients with this change. The host ends the attempt at the existing 60 second `hostConfirmTimeout`. The joiner exits with `safety code handshake failed: timed out` when the exchange does not complete 60 seconds after `connected`.
- The safety code is shown after the DataChannel opens instead of right after `connected`.
- For the same fingerprints, the code is different from before, because it now includes the nonces.

## Dependencies
None. `crypto/rand`, `crypto/sha256`, `crypto/subtle` and `encoding/hex` are in the standard library.

## Acceptance criteria
- [x] Honest host and joiner show the same code and open the chat after both answer `y`.
- [x] The code changes when either nonce or either fingerprint changes.
- [x] A reveal that does not match the commitment ends the attempt with `safety code commitment mismatch` on both roles; the host keeps waiting for new requests.
- [x] Out-of-order, repeated, non-hex or wrong-length handshake messages end the attempt.
- [x] The host never sends its nonce before it receives the commitment.
- [x] `lightrdp:safety-code-confirmed` sent before the exchange is done does not count as a confirmation.
- [x] Events from a dropped attempt are ignored.
- [x] The joiner exits with `safety code handshake failed: timed out` when the exchange does not complete 60 seconds after `connected`.
- [x] Docs, error pages, `README.md` and `CHANGELOG.md` are updated.

## Test plan
- Automated tests:
  - `safety_code_test.go`: same inputs give the same code on both roles; changing any fingerprint or nonce changes the code; `NNN NNN` format; `verifyCommitment` accepts the right nonce and refuses any other; `newNonce` returns 32 bytes and different values on each call.
  - `safety_handshake_test.go`: the joiner and host steps in order produce the same nonces on both sides; a wrong reveal gives the commitment error; out-of-order, repeated, non-hex and wrong-length messages give the handshake error; the host does not produce its nonce before the commitment.
  - `chat_test.go`: a confirmation sent before the exchange is done does not open the chat.
  - `session_test.go`: the existing host and join tests (`TestHostAndJoinConnect`, refusal tests) still pass with the new flow; a nonce event from a dropped attempt is ignored; a handshake error on the host keeps it waiting.
- Manual tests: run `lightrdp server`, `lightrdp host` and `lightrdp join` on one machine, check that the codes match and the chat opens; answer `n` on each side and check the existing refusal messages.

## Open questions
- [x] Joiner timeout: 60 seconds after `connected`, then `safety code handshake failed: timed out`.
- [x] Version: the change goes under `[Unreleased]` in `CHANGELOG.md`; the tag is chosen at release time.
- [x] Error messages: `safety code commitment mismatch` and `safety code handshake failed: <detail>`.
