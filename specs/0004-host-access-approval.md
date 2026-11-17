# 0004: Host access approval

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-16 |
| Pull request | https://github.com/ArthurLoffi/LightRDP/pull/8 |

## Goal
Make the host explicitly confirm each connection request before any WebRTC negotiation or data exchange starts.

## Context
Today the host accepts the first `connect` it receives: `onConnect` (`cmd/client/session.go`) creates the peer, opens the `chat` DataChannel and sends an offer with no check. Anyone who can reach the signaling server and knows (or guesses) the host ID gets a session. `docs/app/getting_started.md` already warns: "No access password yet. Anyone on your network who knows the host ID can connect to it."

With this change, the person at the host decides who connects. The signaling server only sees whether the host accepted or refused: a refusal is the existing `reject` message, and an acceptance is the `offer` that follows. No secret travels through the server.

## Scope
### In scope
- An interactive confirmation on the host (accept or refuse) for every connection request.
- A 60 second timeout: if the host does not accept in time, the request is refused.
- Refusal messages sent back to the client with the existing `reject` message.
- Only `cmd/client` and documentation. The signaling protocol and the signaling server stay unchanged.

### Out of scope
- Access codes, passwords or any other secret.
- TLS (`wss://`) for the signaling server, and the server listening on all interfaces by default. Separate spec.
- User accounts and unattended access.
- Rate limiting on the server.
- Permission levels per session (view only, control, clipboard).

## Requirements
1. On `connect`, the host shows the requester ID and asks for confirmation (`s` / `n`).
2. No peer is created, no offer is sent and no DataChannel is opened until the host accepts.
3. If the host refuses, the client receives `reject` with the message `recusado pelo host` and exits.
4. If the host does not answer within 60 seconds, the request is refused with `reject` and the message `tempo esgotado aguardando o host`, and the client exits.
5. Any answer other than `s` or `S` (including an empty line) refuses the request.
6. While a request is waiting for confirmation, other `connect` requests are refused with the existing `host ocupado`.
7. Offers, answers and candidates from a requester that was not accepted are ignored.
8. The requester ID shown in the prompt is passed through `sanitize`.
9. After a refusal or a timeout, the host keeps waiting for new requests (see Open questions).
10. An answer only counts for the prompt it answers: lines read before the prompt was printed are ignored, and lines read less than 1 second after it are ignored with `Resposta ignorada: rápida demais. Aceitar? [s/N] `. This stops a late answer meant for an earlier request, or text typed before the prompt, from accepting a new requester.

## User experience
Host:
```
go run ./cmd/client -mode host -id HOST -server ws://<SERVER_IP>:8080/ws
Conectado ao servidor como "HOST"
Aguardando conexões...

Pedido de conexão de "FO3TFX". Aceitar? [s/N] s
```
- Answer typed less than 1 second after the prompt: `Resposta ignorada: rápida demais. Aceitar? [s/N] `
- After a refusal: `Pedido de "FO3TFX" recusado.`
- After a timeout: `Pedido de "FO3TFX" recusado: tempo esgotado.`

Client:
```
go run ./cmd/client -mode join -peer HOST -server ws://<SERVER_IP>:8080/ws
Conectado ao servidor como "FO3TFX"
Pedido enviado para "HOST"
Aguardando confirmação do host...
```

No new flags.

## Technical design
- `cmd/client/session.go`:
  - `onConnect` no longer starts the peer. It stores the requester as pending (`pendingID`), prints the prompt and starts waiting for the answer without blocking `loop`. The answer and the 60 second timeout arrive back in `loop` through a channel, so `loop` keeps handling other messages (and refusing new `connect` with `host ocupado`).
  - On accept: the current flow (`startPeer`, `CreateDataChannel("chat")`, `CreateOffer`).
  - On refuse or timeout: send `reject` to the requester, clear the pending state, keep waiting.
  - A late answer after the timeout is ignored.
  - `join` prints `Aguardando confirmação do host...` after sending `connect`.
- Confirmation input: read from the existing single `stdin` reader in `chat.go` (chat only starts reading after the DataChannel opens, so there is no conflict). Hide the reader and the timeout behind small fields on `session` so tests can inject answers and use a short timeout.
- `internal/signaling`: no change. `connect` and `reject` are used as they are.
- `internal/server`: no change.

## Errors
| Message | Cause | Solution |
|---|---|---|
| `erro: conexão recusada: recusado pelo host` | The host person answered `n` or pressed Enter. | Confirm with the host person that they expect the connection and try again. |
| `erro: conexão recusada: tempo esgotado aguardando o host` | Nobody accepted on the host within 60 seconds. | Ask the host person to watch the terminal and try again. |

## Breaking changes
- The host no longer accepts connections automatically; someone must confirm in the host terminal within 60 seconds.
- The protocol does not change, so old and new clients still work together. An old host keeps accepting without confirmation.

## Dependencies
None.

## Acceptance criteria
- [x] The host is asked to confirm before any peer, offer or DataChannel is created.
- [x] Answering `s` starts the session as today.
- [x] Answering `n` or an empty line refuses, and the client shows `recusado pelo host`.
- [x] No answer within 60 seconds refuses, and the client shows `tempo esgotado aguardando o host`.
- [x] A second request during a pending confirmation gets `host ocupado`.
- [x] Messages from a requester that was not accepted are ignored.
- [x] After a refusal or timeout, the host accepts a new request.
- [x] Answers read before the prompt, or less than 1 second after it, are ignored.
- [x] `CHANGELOG.md`, `README.md`, `docs/app/getting_started.md` (remove the "No access password yet" warning, describe the confirmation) and `docs/errors/` (two new pages, linked in `docs/errors/README.md`) are updated.

## Test plan
- Automated tests (`cmd/client`, injected answers and short timeout):
  - accept starts the offer;
  - refuse and empty answer send `reject` with `recusado pelo host`;
  - timeout sends `reject` with `tempo esgotado aguardando o host`;
  - a late answer after timeout is ignored;
  - busy while pending;
  - offer/candidate from a non-accepted requester ignored;
  - a new request after a refusal is prompted again;
  - answers read before the prompt or within 1 second after it are ignored, and a late answer meant for a timed-out request does not accept the next one.
  - Update `TestHostAndJoinConnect` to accept through the injected reader.
- Manual tests: two machines on the same network: accept, refuse, no answer for 60 seconds, a third client connecting while the host prompt is open.

## Open questions
- After a refusal or timeout, should the host keep waiting for new requests (proposed), or exit like it does after a session ends?
A: Keep waiting for new requests
