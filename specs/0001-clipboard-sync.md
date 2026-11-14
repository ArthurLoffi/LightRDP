# 0001: Clipboard sync

| Field | Value |
|---|---|
| Status | Draft |
| Author | ArthurLoffi |
| Created | 2026-11-14 |
| Pull request | |

## Goal
Keep the text clipboard of the host and the client in sync during a session, so text copied on one machine can be pasted on the other.

## Context
In a help desk session the helper often needs to send commands, links or error messages to the other machine, or copy them from it. Typing them by hand is slow and error-prone.

## Scope
### In scope
- Plain text only.
- Both directions: host to client and client to host.
- Sync only while a peer-to-peer session is connected.
- Linux, Windows and macOS.

### Out of scope
- Images, files and rich text.
- Clipboard history.
- Sync through the signaling server. Clipboard data must only travel over the peer-to-peer connection.

## Requirements
1. When the local clipboard text changes, send it to the other side.
2. When text arrives from the other side, write it to the local clipboard.
3. Writing received text must not trigger a new send (no echo loop).
4. Ignore text larger than 64 KB, both when sending and receiving.
5. Clipboard sync can be turned off with a flag, and is on by default.
6. Clipboard data uses its own DataChannel (`clipboard`), separate from the chat.
7. A failure to read or write the clipboard must not end the session.

## User experience
- New flag on `cmd/client`: `-clipboard=true|false` (default `true`).
- When text is received, print a short notice, for example `Área de transferência atualizada (42 caracteres)`. Never print the clipboard content.

## Technical design
- New package `internal/clipboard`: watch the local clipboard and write to it, behind a small interface so it can be faked in tests.
- `cmd/client`: open the `clipboard` DataChannel next to `chat`, and connect it to `internal/clipboard`.
- Message format on the channel: raw UTF-8 text.

## Errors
| Message | Cause | Solution |
|---|---|---|
| `área de transferência indisponível: ...` | No clipboard access (for example, no display on Linux). | Run in a desktop session, or use `-clipboard=false`. |

## Breaking changes
None.

## Dependencies
A clipboard library, to be chosen and approved before implementation (for example `golang.design/x/clipboard`).

## Acceptance criteria
- [ ] Text copied on the host can be pasted on the client, and the other way around.
- [ ] Received text does not echo back to the sender.
- [ ] Text over 64 KB is not sent, and received text over 64 KB is ignored.
- [ ] `-clipboard=false` disables sync on that side.
- [ ] Clipboard content is never printed or logged.
- [ ] Clipboard errors do not end the session.
- [ ] The new error is documented in `docs/errors/`, and `docs/app/` explains the feature.

## Test plan
- Automated tests: `internal/clipboard` with a fake clipboard (change detection, echo prevention, size limit).
- Manual tests: two machines on the same network, copy and paste in both directions, then repeat with `-clipboard=false`.

## Open questions
- Which clipboard library to use, given cgo and Linux display (X11/Wayland) requirements?
- Should the host be able to turn off clipboard sync for the client side?
