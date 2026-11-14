# 0002: Application docs

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-14 |
| Pull request | https://github.com/ArthurLoffi/LightRDP/pull/6 |

## Goal
Write the user-facing documentation in `docs/app/`: how to run LightRDP, and what each feature it has today does.

## Context
`docs/app/README.md` only describes how the folder should be organized. There is no usage guide yet, even though the signaling server, the client and the chat already work (#4). `AGENTS.md` requires every feature to be documented in `docs/app/`, and the checklist item for it was left open in #4.

## Scope
### In scope
- A basic usage guide in `docs/app/`:
  - What you need before starting (Go, the same network, firewall).
  - Starting the signaling server.
  - Starting a session as the host (the machine that is accessed).
  - Connecting to a host.
  - Ending a session.
- A `docs/app/chat/` page for the chat.
- A list of the current features, and what is not available yet.
- Links from the guide to the matching pages in `docs/errors/`.
- Update `docs/app/README.md` so it links to the new pages.

### Out of scope
- Changes to code, flags or messages.
- Features that do not exist yet (remote desktop, access password, clipboard sync from spec 0001, history).
- Translating the messages the program prints, which stay in Portuguese.
- Developer documentation (architecture, signaling protocol).

## Requirements
1. Every flag the user can pass is documented with what it does and its default:
   - `cmd/server`: `-addr` (default `:8080`), `-debug` (default `false`).
   - `cmd/client`: `-server` (default `ws://localhost:8080/ws`), `-mode` (`host` or `join`, required), `-id` (random 6-character ID if empty), `-peer` (required with `-mode join`).
2. The guide shows the messages the user sees at each step, exactly as the program prints them, for example `Aguardando conexões...`, `Pedido de conexão de "<ID>"`, `DataChannel 'chat' aberto! Digite mensagens:`.
3. The guide explains the current behavior that a user may not expect:
   - A host accepts only one session at a time. Other requests get `host ocupado`.
   - If the signaling server goes down after the machines are connected, the chat keeps working.
   - There is no access password yet: anyone on the network who knows the host ID can connect.
4. The network section says which ports must be open (TCP on the server port, UDP between the two machines), and that they must only be opened to the local network.
5. The chat page explains how to send messages, how received messages look (`< message`), and that control characters are removed from received messages.
6. Text follows the "For agents" rule in `docs/app/README.md`: plain language, no technical jargon, no repeated text.
7. Every command in the docs works when copied as written, apart from placeholders such as `<SERVER_IP>`.

## User experience
Documentation only. The user reads `docs/app/README.md` and follows the links to the guide and the chat page.

## Technical design
Files to create or change:
- `docs/app/README.md`: keep the "How to document" and "For agents" sections, and add links to the new pages.
- `docs/app/getting_started.md`: requirements, server, host, join, ending a session, network, feature list.
- `docs/app/chat/README.md`: the chat.
- `README.md`: check that the "Documentation" section links to `docs/app/README.md` (it already does).

Sources for the content: `cmd/server/main.go`, `cmd/client/main.go`, `cmd/client/session.go`, `cmd/client/chat.go`, and `docs/errors/`.

## Errors
None. The guide links to the existing pages in `docs/errors/`.

## Breaking changes
None.

## Dependencies
None.

## Acceptance criteria
- [x] `docs/app/getting_started.md` covers requirements, server, host, join, ending a session and network.
- [x] Every flag of `cmd/server` and `cmd/client` is documented with its default.
- [x] Messages quoted in the docs match the code exactly.
- [x] The one-session limit, the missing access password and the chat surviving a server disconnect are explained.
- [x] `docs/app/chat/README.md` exists.
- [x] `docs/app/README.md` links to every new page.
- [x] Links to `docs/errors/` work.
- [x] `CHANGELOG.md` updated.

## Test plan
- Automated tests: none (documentation only).
- Manual tests: follow the guide on two machines on the same network, from a clean terminal, and check that each step and message matches.

## Open questions
All answered on 2026-11-14:
- One file or one page per step? One file: `docs/app/getting_started.md`.
- "How to share resources as the host"? Listed under "Not available yet".
- `go run` or `go build`? `go run ./cmd/...`, same as `README.md`.
- Language? English only. Program messages are quoted in Portuguese, exactly as printed.
