# 0006: Signaling encryption and DTLS fingerprint verification

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-18 |
| Pull request | |

## Goal
Stop an attacker on the network, or a compromised signaling server, from replacing the DTLS fingerprint inside the SDP and sitting in the middle of the peer-to-peer session. Encrypt the signaling traffic with TLS (`wss://`) and make both users check a safety code proving that the DTLS session really ends at the other user's machine.

## Context
The client talks to the signaling server over plain WebSocket (`ws://`, default `-server ws://localhost:8080/ws` in `cmd/client/main.go`), and the server listens with `srv.ListenAndServe()` (`cmd/server/main.go`). Offers and answers are relayed as `webrtc.SessionDescription` payloads (`internal/signaling/message.go`), and every SDP carries an `a=fingerprint:` line with the hash of the peer's DTLS certificate.

WebRTC trusts that fingerprint to authenticate the DTLS handshake. Today:
- Anyone who can see or change traffic between a client and the server (same Wi-Fi, ARP spoofing, a router) can read the SDP and replace the fingerprint and ICE candidates with their own. Each peer then completes DTLS with the attacker, who relays and reads the chat (and, later, the remote desktop stream). Neither user sees any warning.
- The server relays the payload without decoding it, but it can still rewrite it. TLS alone does not protect against a malicious or compromised server, so both protections are needed.

Related: spec 0004 (host access approval). This spec merges the host approval prompt with the safety code check into a single prompt, which changes when the host is asked (see "Breaking changes").

## Scope
### In scope
- TLS on the signaling server: `cmd/server` serves `wss://` with a certificate and key given by flags. TLS is required when `-bind` is not a loopback address.
- TLS on the client: `cmd/client` verifies the server certificate, can trust a custom CA or self-signed certificate given by flag, and refuses `ws://` to any address that is not loopback.
- Mandatory safety code: after the peer connection reaches `connected`, both clients derive the same short code from the two DTLS certificate fingerprints actually used in the handshake. The users compare it by voice or another channel. The chat does not open until both sides confirm.
- Single host prompt: the host approval from spec 0004 and the safety code check become one prompt, shown when the code is ready.
- Documentation of how to create a self-signed certificate with `openssl` in `docs/app/`.
- Errors for TLS, insecure URLs and an unconfirmed safety code, documented in `docs/errors/`.
- `README.md`, `docs/app/getting_started.md`, `docs/app/chat/README.md` and `CHANGELOG.md` updated.

### Out of scope
- Automatic certificates (ACME / Let's Encrypt) and a helper command that creates certificates.
- Access password or PAKE (separate spec).
- End-to-end encryption or signing of the signaling payload inside the WebSocket (beyond TLS).
- Hiding IP addresses from the signaling server or from the other peer (TURN-only mode, filtering host candidates). TLS already hides them from the network between a client and the server. Hiding them from the server and the other peer needs a TURN server, which is a bigger change, so it goes to a separate spec. The merged prompt makes this more relevant (see "Breaking changes").
- Remembering trusted peers between sessions.

## Requirements
1. `cmd/server` gets `-cert` and `-key` flags. When both are set, it serves TLS (`ListenAndServeTLS`) on the same `/ws` path.
2. `cmd/server` exits with an error when only one of `-cert` and `-key` is set.
3. `cmd/server` exits with an error when `-bind` is not a loopback address (`127.0.0.0/8`, `::1`, `localhost`) and `-cert`/`-key` are not set. `0.0.0.0` and `::` are not loopback.
4. `cmd/client` accepts `wss://` in `-server`. It accepts `ws://` only when the host is a loopback address; any other `ws://` URL is refused before dialing. The default `-server ws://localhost:8080/ws` stays valid.
5. With `wss://`, the server certificate is verified against the system roots, plus the certificates in the file given by a new `-ca` flag when set. Certificate verification is never skipped, and there is no flag to skip it.
6. A failed TLS handshake ends the client with an error that names the server URL and the TLS cause.
7. On a connection request, the host prints `Connection request from "<ID>"` and starts the peer negotiation without asking yet. The 60-second timeout of spec 0004 starts at the request and covers the negotiation and the answer.
8. After the peer connection state becomes `connected`, each client reads the local DTLS certificate fingerprint and the remote certificate presented in the DTLS handshake (not the one written in the SDP), and derives the same safety code from the two fingerprints in a fixed order (host first, joiner second).
9. The safety code is 6 decimal digits, shown in two groups of 3 (for example `482 913`).
10. The host shows a single prompt with the requester ID and the code: `Safety code: 482 913` then `Accept "<ID>" with this code? [y/N]`. The joiner shows `Safety code: 482 913` then `Same code on the host? [y/N]`.
11. Only `y` or `Y` confirms. Any other answer closes the peer connection. The host sends `reject` with `refused by host` and keeps waiting for new requests. The joiner sends `reject` with `safety code not confirmed` and ends.
12. The early-answer rules of spec 0004 apply to both prompts: answers typed before the prompt, or less than 1 second after it, are ignored with `Answer ignored: too fast.` and the prompt repeated.
13. Chat input is not sent and received chat messages are not printed until both sides have confirmed. Each side tells the other it confirmed over the DataChannel, so neither side opens the chat while the other is still deciding.
14. The joiner accepts `reject` from the host while the code is pending, even though the peer connection is already `connected`. Today `reject` is ignored after `connected` (`cmd/client/session.go`).
15. Requests that arrive while the host is negotiating or deciding still get `host busy`.
16. No new signaling message types or fields.

## User experience
Create a self-signed certificate (documented in `docs/app/`), with the server IP in the certificate:
```bash
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 365 \
  -keyout server.key -out server.crt -subj "/CN=LightRDP" \
  -addext "subjectAltName=IP:<SERVER_IP>"
```
Server with TLS:
```bash
go run ./cmd/server -bind <SERVER_IP> -cert server.crt -key server.key
```
Clients (copy `server.crt` to each computer; it is public, `server.key` is not):
```bash
go run ./cmd/client -mode host -id HOST -server wss://<SERVER_IP>:8080/ws -ca server.crt
go run ./cmd/client -mode join -peer HOST -server wss://<SERVER_IP>:8080/ws -ca server.crt
```
On the host:
```
Connection request from "<YOUR_ID>"
Peer connection state changed: connected
Safety code: 482 913
Accept "<YOUR_ID>" with this code? [y/N]
```
On the joiner:
```
Peer connection state changed: connected
Safety code: 482 913
Same code on the host? [y/N]
```
The two people compare the code by phone or in person. When both answer `y`, the chat opens on both sides.

## Technical design
- `cmd/server/main.go`: new `-cert` and `-key` flags and a validation function (loopback check with `net.ParseIP(...).IsLoopback()` plus `localhost`). `main` calls `ListenAndServeTLS(cert, key)` when TLS is configured. The listen log line gets a `tls` attribute. `newServer` unchanged.
- `internal/transport/ws.go`: `Dial` takes a `*tls.Config` (passed through `websocket.DialOptions.HTTPClient`). It refuses `ws://` to non-loopback hosts before dialing. `coder/websocket` already supports `wss://`.
- `cmd/client/main.go`: new `-ca` flag, loads the PEM file into a copy of the system `x509.CertPool`.
- `internal/peer/peer.go`: new method that returns the local and remote DTLS certificate fingerprints after the connection is up, from `pc.SCTP().Transport()` (`DTLSTransport.GetRemoteCertificate`) and the local certificate.
- New file `cmd/client/safety_code.go`: derives the code from the two fingerprints (SHA-256 over host fingerprint + joiner fingerprint, reduced to 6 decimal digits and grouped as `NNN NNN`). Pure function, unit tested.
- `cmd/client/session.go`: `onConnect` starts the negotiation (current `acceptPending` work) without asking. A new state between `connected` and chat open prints the code and waits for the answer on the shared stdin line reader. `reject` is accepted in that state.
- `cmd/client/chat.go`: the DataChannel does not forward stdin or print messages until both confirmations are done. The confirmation message on the DataChannel is not shown as chat text.
- Uses only the Go standard library (`crypto/tls`, `crypto/x509`, `crypto/sha256`) and existing pion APIs.

## Errors
| Message | Cause | Solution |
|---|---|---|
| `error: transport: insecure server URL ws://<ADDRESS>/ws: use wss://` | `-server` uses `ws://` with an address that is not loopback. | Start the server with `-cert` and `-key` and use `wss://`. |
| `error: transport: dial wss://<ADDRESS>/ws: ... x509: certificate signed by unknown authority` | The server uses a self-signed or private CA certificate and the client does not trust it. | Pass the certificate or CA with `-ca <FILE>`. |
| `error: transport: dial wss://<ADDRESS>/ws: ... x509: certificate is valid for <NAME>, not <ADDRESS>` | The certificate does not include the address used in `-server`. | Create the certificate with the server IP or name in its `subjectAltName`. |
| `error: -ca: <CAUSE>` | The `-ca` file does not exist or has no valid PEM certificate. | Check the path and the file content. |
| Server: `-cert and -key must be used together` | Only one of the two flags was given. | Pass both flags. |
| Server: `-cert and -key are required when -bind is not a loopback address` | The server would listen on the network without TLS. | Pass `-cert` and `-key`, or use `-bind 127.0.0.1`. |
| Server: `server stopped err=... tls: ...` | The certificate or key file is missing, unreadable, or they do not match. | Check the files and that the key belongs to the certificate. |
| `error: safety code not confirmed` (joiner refused the code) / `error: connection refused: safety code not confirmed` (host side when the joiner refused) | One user did not confirm the code, or the codes were different (possible interception). | Check that both computers use the right server, then try again. If the codes differ again, do not use that network or server. |

The existing `refused by host` and `timed out waiting for host` errors keep their text; their docs (`docs/errors/refused_by_host.md`, `docs/errors/confirmation_timeout.md`) are updated for the new prompt.

## Breaking changes
- **Server:** `-bind` with a non-loopback address no longer works without `-cert` and `-key`. Commands like `go run ./cmd/server -bind <SERVER_IP>` from the current docs stop working.
- **Client:** `ws://` to a non-loopback address is refused. Every `-server ws://<SERVER_IP>:8080/ws` in the current docs must change to `wss://`.
- **Host approval (spec 0004):** the host is no longer asked before the negotiation. The prompt moves to after `connected`, merged with the safety code, and its text changes from `Accept? [y/N]` to `Accept "<ID>" with this code? [y/N]`. The guarantee "nothing is exchanged between the two computers before the host accepts" is removed: any requester who knows the host ID now gets the host's ICE candidates (IP addresses) and a DTLS connection before the host decides. No chat data passes before both confirm.
- **Chat:** the chat no longer opens right after `connected`; both users must confirm the code first.
- **Internal:** `transport.Dial` signature changes (only used by `cmd/client`).

## Dependencies
None.

## Acceptance criteria
- [x] The server serves `wss://` with `-cert` and `-key`.
- [x] The server refuses to start with only one of `-cert`/`-key`, and with a non-loopback `-bind` without them; it still starts on `127.0.0.1` without them.
- [x] The client refuses `ws://` to non-loopback addresses before dialing, and accepts `ws://localhost`/`ws://127.0.0.1`.
- [x] The client connects over `wss://` to a server with a self-signed certificate when `-ca` is given, and fails with a TLS error when it is not.
- [x] Both clients print the same safety code for the same session, and different codes when the DTLS certificate of either side changes.
- [x] The host sees a single prompt with the ID and the code; no `Accept? [y/N]` before negotiation.
- [x] No chat message is sent or printed before both users confirm.
- [x] A refusal on either side closes the connection on both sides with the documented messages; the host keeps waiting for new requests.
- [x] Early answers and the 60-second timeout follow the documented behavior; `host busy` still works.
- [x] Every new error is documented in `docs/errors/` and linked from `docs/errors/README.md` and `docs/app/getting_started.md`.
- [x] `docs/app/` explains how to create the certificate with `openssl` and which file to copy.
- [x] `README.md`, `docs/app/` and `CHANGELOG.md` are updated.

## Test plan
- Automated tests:
  - `cmd/server`: TLS on with both flags; error with only one; error with non-loopback `-bind` and no TLS (`0.0.0.0`, `::`, a LAN IP); loopback without TLS accepted; `wss://` handshake with a test certificate.
  - `internal/transport`: `ws://` to a non-loopback host refused without dialing; `ws://127.0.0.1` accepted; `wss://` with a trusted CA succeeds; with an untrusted certificate fails with an x509 error.
  - `cmd/client`: `-ca` loading (valid, missing, invalid PEM).
  - `safety_code`: same inputs give the same code on both roles; any change in either fingerprint changes the code; output is 6 digits in the `NNN NNN` format.
  - `internal/peer`: two local peers get matching fingerprints (local of A equals remote of B).
  - Session: single host prompt after `connected`; chat blocked until both confirm; `n` on host sends `refused by host` and keeps waiting; `n` on joiner sends `safety code not confirmed`; `reject` accepted while the code is pending; early answers ignored; timeout covers negotiation plus answer; `host busy` during negotiation.
- Manual tests:
  - Two machines on the LAN through a `wss://` server with a self-signed certificate made with the documented `openssl` command; compare codes and chat.
  - Client without `-ca` against that server: TLS error shown.
  - Client with `ws://<SERVER_IP>`: refused.
  - Server with `-bind <SERVER_IP>` and no certificate: refused.
  - Answer `n` on each side in turn: both sides end the session as documented.

## Open questions
- None. Decided: the safety code is 6 digits; the IP exposure caused by the merged prompt is accepted until a separate spec handles it.
