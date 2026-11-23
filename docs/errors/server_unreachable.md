# Cannot reach the signaling server
**Message:** `error: transport: dial wss://<ADDRESS>/ws: ...` (or `ws://` on the same computer)

## Description
The client could not connect to the signaling server.

## Cause
- The server is not running.
- The `--server` address or port is wrong.
- The server listens on `127.0.0.1` (the default), so it only accepts connections from its own machine.
- A firewall is blocking the server port.

## Possible solution
- Start the server with `go run ./cmd/lightrdp server`.
- Check the address: it must be `wss://<SERVER_IP>:<PORT>/ws`. Use the server's network IP, not `localhost`, when connecting from another machine. `ws://` only works on the same computer.
- When connecting from another machine, start the server with `--bind <SERVER_IP> --cert server.crt --key server.key`. See [Security](../app/security/README.md).
- If the message ends with an `x509: ...` error, see [Server certificate not trusted](untrusted_certificate.md) or [Certificate does not match the server address](certificate_name_mismatch.md).
- Allow the server port (default `8080`) in the server machine's firewall, for the local network only.

_Documented on: 2026-11-13 (updated 2026-11-21)_
