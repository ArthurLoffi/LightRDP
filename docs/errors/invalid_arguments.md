# Invalid arguments
**Messages:**
- `usage: client -mode host|join [-server ws://IP:8080/ws] [-id ID] [-peer HOST_ID]`
- `-peer is required with -mode join`

## Description
The client was started with missing or invalid options.

## Cause
- `-mode` is missing or is not `host` or `join`.
- `-mode join` was used without `-peer`.

## Possible solution
- Start the host with `-mode host`.
- Start the other machine with `-mode join -peer <HOST_ID>`.

_Documented on: 2026-11-13_
