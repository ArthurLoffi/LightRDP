# Host did not answer in time
**Message:** `error: connection refused: timed out waiting for host`

## Description
The host did not accept your connection request within 60 seconds.

## Cause
The 60 seconds start when the request arrives at the host. They cover connecting the two computers and the host's answer to `Accept "<YOUR_ID>" with this code? [y/N]`. If the host does not answer in time, or the computers take too long to connect, the request is refused.

## Possible solution
- Ask the person at the host to watch the host terminal.
- Try again.

_Documented on: 2026-11-16 (updated 2026-11-18)_
