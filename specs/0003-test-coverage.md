# 0003: Test coverage for existing code

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-14 |
| Pull request | |

## Goal
Add automated tests for every package that has none today, and more tests for `internal/peer`, changing only `cmd/server/main.go` so the server setup can be tested.

## Context
`go test ./...` only runs tests in `internal/peer`. Every other package reports `[no test files]`:

- `internal/signaling`
- `internal/transport`
- `internal/server`
- `internal/config`
- `cmd/client`
- `cmd/server`

`AGENTS.md` says every feature must have tests. The signaling server, the client and the chat were merged in #4 without tests for these packages. This spec covers that gap.

## Scope
### In scope
- New `*_test.go` files for every package listed in "Context".
- A refactor of `cmd/server/main.go` that moves the logger and HTTP server setup out of `main()`, with no change in behavior.
- New tests in the existing `internal/peer/peer_test.go`.
- Tests in the same package as the code (`package main` for `cmd/client`).
- Tests use only the standard library (`testing`, `net/http/httptest`, `log/slog`, `encoding/json`) and the dependencies already in `go.mod`.
- A `CHANGELOG.md` entry.

### Out of scope
- Changes to existing `.go` files that are not tests, except `cmd/server/main.go`, and `cmd/client/chat.go` only if needed (see requirement 1).
- Fixing bugs the new tests find. They are listed in the pull request description, not fixed here.
- Tests for `main()` itself in `cmd/client/main.go` and `cmd/server/main.go` (flag parsing, `os.Exit`, signals). `cmd/client/main.go` is not changed.
- Coverage targets or CI setup.

## Requirements
1. The only non-test files that may change are `cmd/server/main.go` (requirement 7) and `cmd/client/chat.go`, the latter only if `setupChannel` cannot be tested otherwise. The expected design does not change `chat.go` (see "Technical design"). `git diff master --stat` lists only `*_test.go` files, `CHANGELOG.md`, `cmd/server/main.go` and, if needed, `cmd/client/chat.go`.
2. Tests follow the style of `internal/peer/peer_test.go`: same package (white-box), `t.Helper()` in helpers, `t.Cleanup` for teardown, timeouts instead of waiting forever.
3. Tests never weaken assertions or special-case test code (`AGENTS.md`).
4. Any test server listens only on loopback (`httptest.NewServer`, which binds `127.0.0.1`). No test opens a port to the network.
5. `go test ./...` and `go test -race ./...` pass, except tests that show a bug. Those tests assert the correct behavior and stay failing: no `t.Skip`, no weakened assertion. Each one is listed in the pull request description.
6. Tests that wait on network events use timeouts so a failure does not hang the run.
7. The `cmd/server/main.go` refactor keeps the same flags (`-addr`, `-debug`), defaults, log messages, `/ws` path, `ReadHeaderTimeout` (10 s) and shutdown timeout (5 s).

### Behavior to cover

`internal/signaling` (`message_test.go`):
- `Relayable` returns `true` for `connect`, `offer`, `answer`, `candidate`, `reject`, and `false` for `register`, `error` and an unknown type.
- `New` with a `nil` payload leaves `Payload` empty; with a payload it stores the JSON; with a payload that cannot be marshaled (for example a `chan`) it returns an error that wraps the marshal error.
- `NewError` sets `Type` to `error`, an empty `From`, the given `To`, and a payload that decodes to the given text.
- `Decode` returns an error for an empty payload and for an invalid payload, and decodes a valid one.
- JSON encoding of `Message` omits `from`, `to` and `payload` when empty.

`internal/peer` (added to `peer_test.go`):
- `CreateDataChannel` returns a channel with the given label, and the other peer receives it through `OnDataChannel` after connecting.
- `OnICECandidate` is called with at least one candidate after `CreateOffer`, and never with an empty candidate.
- `HandleOffer` with an invalid SDP returns an error that starts with `peer: set remote`.
- `HandleAnswer` without a local offer returns an error.
- Candidates queued before the remote description are applied by `HandleOffer` and `HandleAnswer`, and the peers connect.
- `CreateOffer` on a closed peer returns an error that starts with `peer: create offer`.

`internal/config` (`peer_config_test.go`):
- `NewPeerConfig` copies `ICEServers` into the `webrtc.Configuration`, including when the list is empty.

`internal/server` (`hub_test.go`, `handler_test.go`), using `httptest.NewServer(hub)` and real WebSocket clients:
- A client registers and a second client can send it a relayable message.
- Registering with an empty ID returns an `error` message with `id vazio`.
- Registering an ID already in use returns `id já registrado`.
- Registering twice on the same connection returns `conexão já registrada`.
- Sending before registering returns `registre-se antes de enviar mensagens`.
- Sending `register`-only or `error` types to another client returns `tipo de mensagem não permitido: <type>`.
- Sending to an unknown ID returns `destino não encontrado: <id>`.
- The server overwrites `From` with the sender's registered ID, so a client cannot impersonate another.
- The payload is relayed unchanged.
- After a client disconnects, its ID is removed and can be registered again.
- `unregister` does not remove an ID that now belongs to a different connection.
- A message larger than 64 KiB closes the connection.
- A plain HTTP request (no WebSocket upgrade) to the handler does not crash the hub.

`internal/transport` (`ws_test.go`), against a loopback `httptest` WebSocket server:
- `Dial` to an unreachable URL returns an error that starts with `transport: dial`.
- `Send` writes a message the server receives; `Messages` delivers messages sent by the server.
- When the server closes the connection, `Messages` is closed and `Err` returns non-nil.
- `Close` closes the connection and the `Messages` channel.
- Concurrent `Send` calls do not race (`-race`).

`cmd/server` (`main_test.go`):
- `newLogger(false)` does not log `Debug` and logs `Info`; `newLogger(true)` logs `Debug`.
- `newServer` sets `Addr` to the given address and `ReadHeaderTimeout` to 10 s.
- With `httptest.NewServer(srv.Handler)`, a WebSocket client can connect to `/ws` and register; any other path returns `404`.

`cmd/client` (`chat_test.go`, `session_test.go`):
- `setupChannel`, on two connected peers: a line read from `stdin` is sent over the channel after it opens; blank lines are not sent; leading and trailing spaces are trimmed; a received message is printed as `< <text>` after `sanitize`.
- `sanitize` keeps normal text and tabs, and removes control characters (`\r`, `\n`, `\x1b`, ANSI sequences start), and bidi characters `U+202A`–`U+202E` and `U+2066`–`U+2069`.
- `session.handle`, with a `transport.Conn` connected to a real `server.Hub` on loopback:
  - `offer`, `answer`, `candidate` and `reject` from an ID that is not the current remote are ignored.
  - `error` with a non-empty `From` is ignored.
  - `error` from the server before connecting ends the session with `servidor: <message>`.
  - `reject` from the remote before connecting ends the session with `conexão recusada: <message>`.
  - An unknown type does not end the session.
  - A second `connect` while a peer exists sends `reject` with `host ocupado` to the requester.
- End to end: a host session and a join session on the same hub reach `PeerConnectionStateConnected`.
- `loop` returns `nil` when the context is cancelled.
- `loop` returns `servidor de sinalização desconectado: ...` when the server closes before the peer connects.

## User experience
None. Developers run `go test ./...` as today.

## Technical design
`cmd/server/main.go` refactor (same package, private functions):
- `newLogger(debug bool) *slog.Logger`: the current level logic and `slog.NewTextHandler(os.Stderr, ...)`.
- `newServer(addr string, log *slog.Logger) *http.Server`: the current mux with `/ws` on `server.NewHub(log)`, `Addr` and `ReadHeaderTimeout`.
- `main()` keeps flag parsing, signals, `ListenAndServe` and `Shutdown`, and calls the two functions.

Test files:

| File | Package |
|---|---|
| `internal/signaling/message_test.go` | `signaling` |
| `internal/config/peer_config_test.go` | `config` |
| `internal/server/hub_test.go` | `server` |
| `internal/server/handler_test.go` | `server` |
| `internal/transport/ws_test.go` | `transport` |
| `internal/peer/peer_test.go` (existing) | `peer` |
| `cmd/server/main_test.go` | `main` |
| `cmd/client/chat_test.go` | `main` |
| `cmd/client/session_test.go` | `main` |

Shared helpers stay inside each package's test files (for example a `newTestHub(t)` that returns an `httptest.Server` and its `ws://` URL, and a `dialClient(t, url)` that uses `websocket.Dial` + `wsjson`). Logs go to `slog.New(slog.NewTextHandler(io.Discard, nil))`.

`setupChannel` is tested without changing `cmd/client/chat.go`. Swapping the global `stdin` from a test races with the reader goroutine `setupChannel` starts, and two channels in one process (host and join in the end-to-end test) would share the same `bufio.Reader`. So `TestSetupChannel` and `TestHostAndJoinConnect` run the code in a child process (`TestHelperProcess`, selected by the `LIGHTRDP_TEST_HELPER` environment variable). The parent writes to the child's stdin and reads its stdout, and fails if the child reports a data race.

Tests that need a peer connection use an empty `webrtc.Configuration` (no STUN), like `internal/peer/peer_test.go`, so they do not need internet access.

## Errors
None. No new errors the user can see.

| Message | Cause | Solution |
|---|---|---|
| | | |

## Breaking changes
None. The `cmd/server/main.go` refactor does not change flags, defaults, messages or the `/ws` endpoint.

## Dependencies
None.

## Acceptance criteria
- [x] Every package in "Context" has at least one test file.
- [x] `cmd/server` runs as before: same flags, defaults, log messages and `/ws` path.
- [x] `internal/peer/peer_test.go` has the new tests listed under "Behavior to cover".
- [x] Every item under "Behavior to cover" has a test.
- [x] `go test ./...` and `go test -race ./...` pass, apart from tests that show a bug.
- [x] Tests that show a bug stay failing (no `t.Skip`), are listed in the pull request description, and the bug is not fixed.
- [x] `git diff master --stat` shows only `*_test.go` files, `CHANGELOG.md`, `cmd/server/main.go` and, if needed, `cmd/client/chat.go`.
- [x] `CHANGELOG.md` has an entry under `[Unreleased]` with the date.

## Test plan
- Automated tests: the tests listed in "Behavior to cover", run with `go test ./...` and `go test -race ./...`.
- Manual tests: start `go run ./cmd/server` and check the startup log and that a client can still connect and chat. Run `go test -count=3 ./...` once to check the network tests are not flaky.

## Open questions
- None.
