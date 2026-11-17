# Host busy
**Message:** `erro: conexão recusada: host ocupado`

## Description
The host refused the connection because it is already in a session with someone else, or is already deciding on another connection request.

## Cause
A host accepts only one session at a time, and handles only one connection request at a time.

## Possible solution
- Wait for the current session to end and try again.
- Ask the person on the host machine to restart the host.

_Documented on: 2026-11-13 (updated 2026-11-16)_
