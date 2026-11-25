# Safety code exchange failed
**Messages:**
- `error: safety code handshake failed: <DETAIL>` (on the client)
- `Request from "<ID>" dropped: safety code handshake failed: <DETAIL>` (on the host)

`<DETAIL>` is one of `timed out`, `unexpected commit`, `unexpected nonce`, `unexpected reveal`, `invalid commit`, `invalid nonce`, `invalid reveal` or `unknown message`.

## Description
Before the safety code is shown, the two computers exchange random values in a fixed order. The exchange did not complete, so the connection was closed before the safety code was shown.

## Cause
- `timed out` (client only): the host did not complete the exchange within 60 seconds after `Peer connection state changed: connected`. Usually the host runs an older version of LightRDP that does not know this exchange.
- Any other detail: a message of the exchange was malformed, repeated or sent out of order. Usually the other computer runs a different version of LightRDP. It can also mean that someone between the two computers changed the messages.

## Possible solution
- Make sure both computers run the same LightRDP version (`lightrdp version`), then try again.
- If both run the same version and it happens again, check the server address and certificate, and do not use that network or that server.

After this error, the host keeps waiting for new requests.

_Documented on: 2026-11-24_
