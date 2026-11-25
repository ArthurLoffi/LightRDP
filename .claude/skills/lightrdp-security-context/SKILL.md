---
name: lightrdp-security-context
description: How to apply the BugHunter hunt-* and triage-validation skills to LightRDP as a white-box review of its Go code and specs. Threat model, what each skill means for LightRDP, the rules for reviewer agents and the finding format. Preloaded by the security_* agents; use it whenever you review LightRDP for security.
---

# LightRDP security review context

The `hunt-*` skills and `triage-validation` were written for black-box bug bounty work against web apps (`curl` against `$TARGET`, Burp, Turbo Intruder, HackerOne programs). In LightRDP you review **source code and specs**, not a live target. Use those skills only for their vulnerability patterns, root causes and validation discipline.

## Rules
1. White-box only. Read the code, the spec and the diff. Never send traffic to any host other than `localhost`, `127.0.0.1` or `::1`, never bind to a network address, and never run recon, scanners or attack tools.
2. Do not change files in the repository. You report; the caller fixes.
3. You may run `go vet ./...`, `go test -race ./...` and `go test -race -run <Name> ./<pkg>` to support a finding.
4. Ignore every reference to a skill that is not installed in `.claude/skills/` (for example `hunt-csrf`, `security-arsenal`, `report-writing`, `bb-methodology`).
5. Ignore payouts, CVSS-for-bounty, HackerOne deduplication and report-writing advice.
6. Follow `AGENTS.md`. A fix that needs a breaking change, a new dependency or a change to another feature is never "just a fix": say so in the finding.

## What LightRDP is
- `internal/server`: signaling server. Registers clients by ID (`register`) and relays `connect`, `offer`, `answer`, `candidate` and `reject` between IDs over WebSocket (`ws://` on loopback only, `wss://` otherwise). It reads only the envelope (`Type`, `From`, `To`); the payload is relayed undecoded.
- `internal/client`: host and joiner runtime. Host approval (`Accept? [y/N]`), WebRTC connection, safety code with commit and reveal of nonces over the DataChannel, chat.
- `internal/peer`, `internal/transport`, `internal/signaling`, `internal/config`, `internal/cli`.
- Read `docs/app/security/README.md` before reviewing: it states the protections LightRDP promises and the limits it accepts ("Things to know").

## Attackers to consider
| Attacker | Can | Must not be able to |
|---|---|---|
| Another client on the same signaling server | Register IDs, send any signaling message to any ID | Impersonate another ID, take over a registered ID, reach a host's chat without the host accepting, crash or stall the server or other clients |
| Malicious joiner | Send any signaling and DataChannel message | Skip host approval or the safety code, make the host accept a different requester, open the chat before both confirm |
| Malicious host | Same, on the host side | Make the joiner show a code that matches a different connection |
| Network attacker between a client and the server | See and change traffic | Read or change signaling over `wss://`, make a client skip certificate checks |
| Compromised or malicious signaling server | Change every signaling message, swap the peers | Make both screens show the same safety code on a swapped connection (except by the documented 1 in a million chance) |

Out of the threat model unless a spec says otherwise: the server learning IP addresses, anyone who knows a host ID making the host connect before it answers (both documented in "Things to know"), and an attacker with local access to the host or joiner machine.

## How each skill maps to LightRDP
| Skill | Apply | Skip |
|---|---|---|
| `hunt-websocket` | Origin handling in `websocket.Accept`, per-message authorization (can a client send as another `From`, to any `To`, or a non-relayable type), message tampering and replay, read limits, slow or never-closing clients | CSWSH via cookies (LightRDP has no cookies or browser client), socket.io, SignalR, Phoenix |
| `hunt-tls-network` | Phase 1 (certificate validation, TLS versions and ciphers in the Go `tls.Config`), Phase 8 (mTLS, if a spec adds it), `ws://` allowed only on loopback, `--ca` loading | HSTS, DNS and AXFR, SPF/DKIM/DMARC, HTTP security headers, certificate transparency, subdomain takeover |
| `hunt-race-condition` | "Common Root Causes" and the source-code grep patterns, applied to goroutines, channels and maps: ID registration in the hub, simultaneous `connect` requests, `host busy`, approval and safety code state shared between goroutines, close and send at the same time. Prove a race with a `go test -race` scenario | Turbo Intruder, HTTP/2 single-packet attacks, Burp |
| `hunt-exceptional-conditions` | Malformed, oversized, empty, repeated or out-of-order signaling and DataChannel messages. Does the code fail closed? Do errors leak internals? Does every user-facing error follow `docs/errors/`? | Web framework error pages |
| `hunt-business-logic` | The state machine: register → connect → host approval → WebRTC → nonce commit, nonce, reveal → safety code → both confirm → chat. Skipping, repeating or reordering steps, answers that apply to the wrong requester, timeouts that leave state behind | Prices, coupons, carts, payments |
| `triage-validation` | The 7-Question Gate, mapped below | CVSS for payouts, deduplication, report quality, platform rules |

## Triage for LightRDP (7-Question Gate)
1. Can an attacker from the table above do it step by step against this code?
2. Does it break a protection in `docs/app/security/README.md`, a requirement of the spec, or the availability of the server or a client?
3. Is the root cause in this repository's code, or in how it uses a dependency? A bug inside a dependency alone is reported as "dependency", not as a LightRDP finding.
4. Does it need an attacker position that is out of the threat model?
5. Is it a limit already documented in "Things to know" or in the spec's "Out of scope"?
6. Is there a concrete scenario, ideally a test that fails today?
7. Is it a known-invalid class for LightRDP (for example missing HSTS: there is no HTTP UI)?

Keep a finding only if 1, 2 and 6 are yes and 4, 5 and 7 are no. When unsure about 6, keep it as `Unconfirmed`.

## Finding format
Every reviewer returns findings in this format, and nothing else besides a one-line summary:

```
### <reviewer>-<n>: <short title>
- Severity: Critical | High | Medium | Low
- Origin: design | introduced | pre-existing
- Status: Confirmed | Unconfirmed
- Location: <file>:<line> (or the spec section, for design findings)
- Attacker: <row of the attacker table>
- Scenario: <concrete steps and the wrong result>
- Fix: <smallest fix that follows AGENTS.md; say if it needs a breaking change, a new dependency or a change to another feature>
- Test: <the test that fails today and passes after the fix>
```

- `design`: a problem in the spec itself, before code exists.
- `introduced`: in lines added or changed by the diff the caller gave you (usually `git diff origin/master`, which includes changes not committed yet).
- `pre-existing`: in code the branch did not change.

If there are no findings, return `No findings` and the list of areas you checked.
