# Host busy
**Message:** `error: connection refused: host busy`

## Description
The host refused the connection because it is already in a session with someone else, or is already connecting to or deciding on another connection request.

## Cause
A host accepts only one session at a time, and handles only one connection request at a time.

## Possible solution
- Wait for the current session to end and try again.
- Ask the person on the host machine to restart the host.

_Documented on: 2026-11-13 (updated 2026-11-18)_
