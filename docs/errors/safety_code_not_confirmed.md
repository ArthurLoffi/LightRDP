# Safety code not confirmed
**Messages:**
- `error: safety code not confirmed` (on the client that refused the code)
- `error: connection refused: safety code not confirmed` (on the host, when the client refused the code)

## Description
The client did not confirm the safety code, so the connection was closed before the chat opened.

## Cause
- The person at the client answered something other than `y` to `Same code on the host? [y/N]`.
- The codes on the two computers were different. This can mean that someone on the network, or the server, is between the two computers.

## Possible solution
- Check that both computers use the right server address and certificate, then try again.
- Compare the code by phone or in person, digit by digit.
- If the codes are different again, do not use that network or that server.

After this error, the host keeps waiting for new requests.

_Documented on: 2026-11-18_
