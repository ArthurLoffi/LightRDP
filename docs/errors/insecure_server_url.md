# Insecure server address
**Message:** `error: transport: insecure server URL ws://<ADDRESS>/ws: use wss://`

## Description
The client refused to connect to the server without encryption.

## Cause
`--server` uses `ws://` with an address of another computer. Without encryption, anyone on the network could read or change the connection data. `ws://` is only allowed for the same computer (`localhost`, `127.0.0.1` or `::1`).

## Possible solution
- Start the server with `--cert` and `--key`. See [Security](../app/security/README.md).
- Use `wss://<SERVER_IP>:8080/ws` in `--server`, and pass the server certificate with `--ca` if it is self-signed.

_Documented on: 2026-11-18 (updated 2026-11-21)_
