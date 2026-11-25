# Safety code commitment mismatch
**Messages:**
- `error: safety code commitment mismatch` (on the client)
- `Request from "<ID>" dropped: safety code commitment mismatch` (on the host)

## Description
Before the safety code is shown, the two computers exchange random values. The client first sends a fingerprint of its value, and reveals the value only after it receives the host's value. The value revealed by the client did not match the fingerprint it sent first, so the connection was closed before the safety code was shown.

## Cause
- Someone on the network, or the server, is between the two computers and changed the messages.
- The other computer runs a modified or broken version of LightRDP.

## Possible solution
- Do not continue with this connection.
- Check that both computers use the right server address and certificate, then try again.
- If it happens again, do not use that network or that server.

After this error, the host keeps waiting for new requests.

_Documented on: 2026-11-24_
