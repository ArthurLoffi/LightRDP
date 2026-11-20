# 0007: Fix flaky TestNewServerRoutesWebSocket

| Field | Value |
|---|---|
| Status | In review |
| Author | ArthurLoffi |
| Created | 2026-11-19 |
| Pull request | https://github.com/ArthurLoffi/LightRDP/pull/12 |

<!--
Status values:
- Draft: still being written, do not implement.
- Approved: ready to be implemented with /implement.
- In review: implemented, pull request open.
- Done: merged. Move the file to specs/done/.
-->

## Goal
Make `TestNewServerRoutesWebSocket` in `cmd/server/main_test.go` pass every time, by replacing its retry loop with one that works with how `coder/websocket` handles read timeouts. Only test code changes, plus a testing rule in `AGENTS.md` so the mistake is not repeated.

## Context
`TestNewServerRoutesWebSocket` (added in spec 0003, #7) fails intermittently. Running it 50 times with `go test ./cmd/server/ -run TestNewServerRoutesWebSocket -count=50` gave 20 failures, all with:

```
main_test.go:121: Expected relayed message before timeout, got: failed to read JSON message: failed to get reader: use of closed network connection
```

The test registers `alice` and `bob` on two separate connections, then loops: alice sends `connect` to `bob`, and the test reads bob's connection with a 200 ms context. If the read fails, it tries again until a 5 s deadline.

There are two problems, and the second turns the first into a failure:

1. **Registration race.** Alice's `register` and `connect` go over the same connection, so they reach the hub in order. Bob's `register` goes over a different connection, handled by a different goroutine. The hub can route alice's `connect` before bob is registered. The hub then answers alice with `error` `target not found: bob` and sends nothing to bob. This is correct server behavior. A probe run with `go test -overlay` (the repo was not changed) confirmed it: in 11 of 30 runs, bob's first read hit `context deadline exceeded` while alice had received `target not found: bob`.
2. **The retry can never succeed.** In `coder/websocket` v1.8.15, when the context passed to a read expires, the library closes the whole connection (`setupReadTimeout` in `conn.go` runs `c.close()` through `context.AfterFunc`). After the first 200 ms timeout, bob's connection is closed. Every later read fails immediately with `use of closed network connection`. The loop keeps sending `connect` until the 5 s deadline and then fails. On the server side, bob's connection also ends, so the hub unregisters bob.

The retry loop was meant to handle problem 1, but problem 2 makes the retry useless.

Other tests checked:
- `internal/server/hub_test.go` avoids the race with `waitRegistered`, which polls `hub.lookup`. `cmd/server` tests cannot do that: `newServer` returns an `*http.Server` and the `Hub` is not reachable.
- `assertNoMessage` in `internal/server/hub_test.go` also reads with a short timeout that closes the connection, but it is always the last step of its subtest, so it is not affected.
- No other test retries a read after a timed-out read.

## Scope
### In scope
- Rewrite the synchronization in `TestNewServerRoutesWebSocket` in `cmd/server/main_test.go`.
- New test helpers in `cmd/server/main_test.go` if needed.
- A new testing rule in the `Conventions` section of `AGENTS.md` (see "Technical design").
- A `CHANGELOG.md` entry under `Fixed`.

### Out of scope
- Any change to non-test code: `cmd/server/main.go`, `internal/server`, `internal/signaling`. The server behaves correctly.
- A registration acknowledgement message in the signaling protocol. That is a contract change and is not needed to fix the test.
- Other tests, including `assertNoMessage` in `internal/server/hub_test.go`.
- CI setup.

## Requirements
1. The only files that change are `cmd/server/main_test.go`, `AGENTS.md`, `CHANGELOG.md` and this spec. `git diff master --stat` lists nothing else.
2. The test keeps its name and what it checks: a `connect` sent by `alice` through the `/ws` route of `newServer` reaches `bob` with `Type == connect` and `From == "alice"`.
3. No read on bob's connection uses a context that can expire before the overall test timeout (`testTimeout`, 5 s). A read context that expires closes the connection, so it must only expire when the test is about to fail anyway.
4. The test still handles the registration race: if alice's `connect` reaches the hub before bob is registered, the test sends `connect` again instead of failing.
5. The test still fails, with a clear message, if bob gets no `connect` within `testTimeout`, or gets a message that is not `connect` from `alice`.
6. No `time.Sleep`-based guesses about when registration finishes, no `t.Skip`, no weakened assertions, no change in server behavior to make the test pass (`AGENTS.md`).
7. Goroutines started by the test end before or at test cleanup (connections are closed by the existing `t.Cleanup` in `dialServer`).
8. The test server keeps listening only on loopback (`httptest.NewServer`).

## User experience
None for app users. Developers see `go test ./...` pass consistently instead of failing about 1 run in 3 on `cmd/server`.

## Technical design
### `cmd/server/main_test.go`
The retry is driven by the server's reply:

- Start one goroutine that reads bob's connection once, with a context bound to `testTimeout`, and sends the result (message or error) on a channel. This read is never interrupted early, so bob's connection stays open.
- Alice sends `connect` to `bob`.
- Start a goroutine that reads alice's connection with a context bound to `testTimeout` and sends the result on a second channel.
- Loop with `select`:
  - Result from bob: check that it is `connect` from `alice` and return. Fail on a read error or any other message.
  - Result from alice: if it is `error` with `target not found: bob` (decoded with `signaling.ErrorPayload`), bob was not registered yet. Send `connect` again and start a new read on alice. Fail on any other message or read error.
  - Overall deadline (`testTimeout`): fail with the last error seen.
- When bob's read succeeds, the pending read on alice stays blocked until the `dialServer` cleanup closes the connection, which ends that goroutine.

Each retry happens only after the server reports that bob is not registered, so the test does not depend on timing, and it checks why it retried. Resending `connect` on a fixed interval was considered and rejected.

### `AGENTS.md`
Add one bullet to `Conventions`, next to the other `Tests:` bullets:

```
- Tests: with `coder/websocket`, a read whose context expires closes the connection. Never retry a read on the same connection after a timeout.
```

## Errors
None. No new user-facing errors; nothing to add to `docs/errors/`.

| Message | Cause | Solution |
|---|---|---|
| | | |

## Breaking changes
None.

## Dependencies
None.

## Acceptance criteria
- [x] `go test ./cmd/server/ -run TestNewServerRoutesWebSocket -count=200` passes with no failures.
- [x] `go test -race ./cmd/server/ -run TestNewServerRoutesWebSocket -count=100` passes with no failures.
- [x] `go test -cpu 1,2,4 ./cmd/server/ -run TestNewServerRoutesWebSocket -count=50` passes with no failures.
- [x] `go test ./...` and `go test -race ./...` pass.
- [x] No read on bob's connection uses a timeout shorter than `testTimeout`.
- [x] `AGENTS.md` has the new `Tests:` bullet about `coder/websocket` read timeouts in `Conventions`.
- [ ] `git diff master --stat` lists only `cmd/server/main_test.go`, `AGENTS.md`, `CHANGELOG.md` and this spec. Not met: on request of the author, the pull request also moves `specs/0005-english-translation.md` and `specs/0006-signaling-encryption.md` to `specs/done/`. No other file changes.
- [x] `CHANGELOG.md` has an entry under `Fixed` with the date and version tag.

## Test plan
- Automated tests: the fixed `TestNewServerRoutesWebSocket` itself, run with the repeat counts in "Acceptance criteria". Before the fix, the same command fails about 40% of the time (20 of 50 runs), so 200 passing runs in a row shows the fix works.
- Manual tests: temporarily break the route (for example, register the hub on a different path in a local, uncommitted change) and check the test fails within `testTimeout` with a clear message instead of hanging. Revert the change afterwards.

## Open questions
- None. Resolved on 2026-11-19: the retry is driven by alice's `target not found: bob` reply (not a fixed interval), and `AGENTS.md` gets the `coder/websocket` read timeout rule.
