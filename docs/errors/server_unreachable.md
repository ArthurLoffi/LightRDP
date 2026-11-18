# Cannot reach the signaling server
**Message:** `error: transport: dial ws://<ADDRESS>/ws: ...`

## Description
The client could not connect to the signaling server.

## Cause
- The server is not running.
- The `-server` address or port is wrong.
- The server listens on `127.0.0.1` (the default), so it only accepts connections from its own machine.
- A firewall is blocking the server port.

## Possible solution
- Start the server with `go run ./cmd/server`.
- Check the address: it must be `ws://<SERVER_IP>:<PORT>/ws`. Use the server's network IP, not `localhost`, when connecting from another machine.
- When connecting from another machine, start the server with `-bind <SERVER_IP>` (or `-bind 0.0.0.0`).
- Allow the server port (default `8080`) in the server machine's firewall, for the local network only.

_Documented on: 2026-11-13_
