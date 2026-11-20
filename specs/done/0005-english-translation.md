# 0005: English translation

| Field | Value |
|---|---|
| Status | Done |
| Author | ArthurLoffi |
| Created | 2026-11-17 |
| Pull request | https://github.com/ArthurLoffi/LightRDP/pull/9 |

## Goal
Make every user-facing text in LightRDP English: client output and prompts, server logs, error messages, documentation, tests and open specs. Translate the existing code comments to English.

## Context
The code, terminal output and server logs are in Portuguese, while the README, `docs/` and most of the specs are already in English. Documentation quotes Portuguese messages (for example `erro: conexão recusada: host ocupado`), so the project is mixed. The code comments are also in Portuguese.

## Scope
### In scope
- `cmd/client`: flag descriptions, usage text, status lines, the host confirmation prompt, chat notices and error messages.
- `cmd/server`: flag descriptions and log messages.
- `internal/server`: log messages and the error texts sent to clients (`errIDTaken`, `errAlreadyNamed`, `errEmptyID`, `errNotRegistered`, `tipo de mensagem não permitido`, `destino não encontrado`).
- `internal/signaling`: error texts (`signaling: %s sem payload`).
- Translate every comment in the Go code (`cmd/`, `internal/`) to English, including tests and package doc comments.
- Tests: expected strings, test data and messages that are in Portuguese.
- `docs/app/`, `docs/errors/`, `README.md`: every quoted message updated to the new English text.
- Open specs: `specs/0001-clipboard-sync.md` (Portuguese example messages).
- `CHANGELOG.md`: new entry under `[Unreleased]`, and rewrite past entries so quoted messages use the new English text.

### Out of scope
- Supporting more than one language at runtime (no i18n framework, no `-lang` flag).
- Behavior changes other than the text (timeouts, flow, protocol message types and fields stay the same).
- Renaming files, packages, functions or variables.
- Specs already in `specs/done/` (0002, 0003, 0004) stay as they are.
- Adding new comments. Comments are only translated; none are added.

## Requirements
1. No Portuguese text remains in `cmd/`, `internal/`, `docs/`, `README.md`, `CHANGELOG.md` or open specs. `specs/done/` is not changed.
2. Every comment in `.go` files is English. No new comments are added.
3. The client prints the English texts from the translation table below.
4. The server logs the English texts from the translation table below. Log keys (`addr`, `path`, `id`, `type`, `err`, `remote`) stay the same.
5. The error texts the server sends to clients, and the `reject` reasons the host sends to joiners, are English.
6. Every documented error in `docs/errors/` shows the new English message.
7. Existing tests keep testing the same behavior, with only their expected strings and data translated.
8. The host accepts a connection only with `y` or `Y`. Any other answer, including `s`, refuses it.

## User experience
Approved translations.

### Client (`cmd/client`)
| Current | New |
|---|---|
| `URL do servidor de sinalização` | `signaling server URL` |
| `host (aguarda conexões) ou join (conecta a um host)` | `host (waits for connections) or join (connects to a host)` |
| `seu ID (gerado se vazio)` | `your ID (generated if empty)` |
| `ID do host (obrigatório em -mode join)` | `host ID (required with -mode join)` |
| `uso: client -mode host\|join [-server ws://IP:8080/ws] [-id ID] [-peer ID_DO_HOST]` | `usage: client -mode host\|join [-server ws://IP:8080/ws] [-id ID] [-peer HOST_ID]` |
| `-peer é obrigatório em -mode join` | `-peer is required with -mode join` |
| `erro: ...` | `error: ...` |
| `Conectado ao servidor como "ID"` | `Connected to server as "ID"` |
| `Aguardando conexões...` | `Waiting for connections...` |
| `Pedido enviado para "HOST"` | `Request sent to "HOST"` |
| `Aguardando confirmação do host...` | `Waiting for host confirmation...` |
| `Pedido de conexão de "ID"` | `Connection request from "ID"` |
| `Aceitar? [s/N] ` | `Accept? [y/N] ` |
| `Resposta ignorada: rápida demais. Aceitar? [s/N] ` | `Answer ignored: too fast. Accept? [y/N] ` |
| `Pedido de "ID" recusado.` | `Request from "ID" refused.` |
| `Pedido de "ID" recusado: tempo esgotado.` | `Request from "ID" refused: timed out.` |
| `host ocupado` | `host busy` |
| `recusado pelo host` | `refused by host` |
| `tempo esgotado aguardando o host` | `timed out waiting for host` |
| `conexão recusada: ...` | `connection refused: ...` |
| `servidor: ...` | `server: ...` |
| `aviso do servidor: ...` | `server notice: ...` |
| `mensagem desconhecida: "TYPE"` | `unknown message: "TYPE"` |
| `Servidor de sinalização desconectado; chat P2P continua.` | `Signaling server disconnected; P2P chat continues.` |
| `servidor de sinalização desconectado: ...` | `signaling server disconnected: ...` |
| `erro ao enviar candidato: ...` | `failed to send candidate: ...` |
| ` DataChannel 'chat' aberto! Digite mensagens: ` | ` DataChannel 'chat' open! Type messages: ` |
| `erro ao enviar: ...` | `failed to send: ...` |

### Server (`cmd/server`, `internal/server`)
| Current | New |
|---|---|
| `endereço de escuta (":8080" escuta em todas as interfaces)` | `listen address (":8080" listens on all interfaces)` |
| `logs detalhados` | `verbose logs` |
| `servidor de sinalização ouvindo` | `signaling server listening` |
| `servidor parou` | `server stopped` |
| `encerrando` | `shutting down` |
| `fila cheia, mensagem descartada` | `queue full, message dropped` |
| `leitura encerrada` | `read ended` |
| `upgrade falhou` | `upgrade failed` |
| `nova conexão` | `new connection` |
| `cliente registrado` | `client registered` |
| `cliente removido` | `client removed` |
| `id já registrado` | `id already registered` |
| `conexão já registrada` | `connection already registered` |
| `id vazio` | `empty id` |
| `registre-se antes de enviar mensagens` | `register before sending messages` |
| `tipo de mensagem não permitido: TYPE` | `message type not allowed: TYPE` |
| `destino não encontrado: ID` | `target not found: ID` |

### Signaling (`internal/signaling`)
| Current | New |
|---|---|
| `signaling: TYPE sem payload` | `signaling: TYPE has no payload` |

## Technical design
- Text-only changes to string literals in:
  - `cmd/client/main.go`, `cmd/client/session.go` (including the `hostBusyMessage`, `refusedByHostMessage`, `confirmTimeoutMessage` constants), `cmd/client/chat.go`.
  - `cmd/server/main.go`.
  - `internal/server/hub.go`, `internal/server/client.go`, `internal/server/handler.go`.
  - `internal/signaling/message.go`.
- `isAcceptAnswer` in `cmd/client/session.go` accepts `y` and `Y` instead of `s` and `S`.
- Translate every `//` comment in `.go` files to English, including tests (for example `internal/peer/peer_test.go`) and package doc comments (`internal/server/hub.go`, `internal/signaling/message.go`, `internal/transport/ws.go`, `internal/peer/peer.go`). Keep each comment short and keep the same meaning. Do not add new comments.
- Update expected strings and data in `cmd/client/session_test.go`, `cmd/client/host_confirmation_test.go`, `cmd/client/chat_test.go`, `internal/server/hub_test.go`, `internal/signaling/message_test.go` and any other test that matches a translated text.
- Update every quoted message in `README.md`, `docs/app/getting_started.md`, `docs/app/chat/README.md` and every file in `docs/errors/`, including `s` to `y` where the accept answer is described.
- Replace the test data `olá, mundo` in `cmd/client/chat_test.go` with English text that still has non-ASCII characters (for example `héllo, wörld`), so the case keeps checking that non-ASCII text passes `sanitize`.
- `CHANGELOG.md`: add the entry under `[Unreleased]` and rewrite past entries that quote Portuguese messages (`Aceitar? [s/N]`, `recusado pelo host`, `tempo esgotado aguardando o host`, `host ocupado`) with the English text.
- Update Portuguese example messages in `specs/0001-clipboard-sync.md` (`Área de transferência atualizada (42 caracteres)`, `área de transferência indisponível: ...`).
- The signaling protocol (message types, JSON fields) does not change. Only the free text inside `ErrorPayload.Message` changes. Clients never parse that text, so old and new clients and servers still work together; they only show text in the other language.

## Errors
No new errors. Existing errors change their message text; each page in `docs/errors/` is updated.

| Message | Cause | Solution |
|---|---|---|
| `error: connection refused: host busy` | Same as `host_busy.md` today. | Same as today. |
| `error: connection refused: refused by host` | Same as `refused_by_host.md` today. | Same as today. |
| `error: connection refused: timed out waiting for host` | Same as `confirmation_timeout.md` today. | Same as today. |
| `error: server: id already registered` | Same as `id_already_registered.md` today. | Same as today. |
| `error: server: target not found: <ID>` | Same as `peer_not_found.md` today. | Same as today. |
| `error: signaling server disconnected: ...` | Same as `server_disconnected.md` today. | Same as today. |
| `error: peer connection failed` | Same as `peer_connection_failed.md` today. | Same as today. |
| `error: transport: dial ...` | Same as `server_unreachable.md` today. | Same as today. |
| `usage: client ...` / `-peer is required with -mode join` | Same as `invalid_arguments.md` today. | Same as today. |

## Breaking changes
Approved by the owner:
- Every message printed by the client and logged by the server changes. Anyone matching the old text (scripts, log filters, alerts) must update.
- The text inside `reject` and `error` payloads changes. The protocol stays compatible, but mixed-version setups show mixed languages.
- The accept key changes from `s`/`S` to `y`/`Y`. Hosts who type `s` refuse the connection instead of accepting it.

## Dependencies
None.

## Acceptance criteria
- [x] `grep` for Portuguese words in `cmd/`, `internal/`, `docs/`, `README.md`, `CHANGELOG.md` and open specs finds nothing (`specs/done/` excluded).
- [x] Every comment in `.go` files is English, and no comment was added.
- [x] The host accepts with `y` and `Y`, and refuses with `s`.
- [x] Client and server print the texts from the translation tables.
- [x] Every page in `docs/errors/` and every table in `docs/app/` shows the English message.
- [x] `go vet ./...` and `go test ./...` pass.
- [x] `CHANGELOG.md` has an entry under `[Unreleased]`, and past entries quote the English messages.

## Test plan
- Automated tests: existing tests updated to the English strings (session refusal, server errors, host confirmation prompt, hub errors, signaling payloads). Tests check that the host accepts with `y` and `Y` and refuses with `s`.
- Manual tests: run the server and two clients, go through accept, refuse, timeout, host busy, host not found and duplicate ID, and check every message and server log line is English and matches the docs.

## Open questions
None. Answered by the owner on 2026-11-17:
- Accept key: `y` and `Y` only.
- Translations: the tables above are approved.
- Comments: keep them, translate to English, do not add more.
- `specs/done/`: not changed.
- `CHANGELOG.md`: rewrite past entries; new entry under `[Unreleased]`.
- `olá, mundo` test data: replaced with English text.
- Spec 0004: moved to `specs/done/` with `Status: Done` (done while writing this spec).
