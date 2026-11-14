# Cannot reach the signaling server
**Message:** `erro: transport: dial ws://<ADDRESS>/ws: ...`

## Description
The client could not connect to the signaling server.

## Cause
- The server is not running.
- The `-server` address or port is wrong.
- A firewall is blocking the server port.

## Possible solution
- Start the server with `go run ./cmd/server`.
- Check the address: it must be `ws://<SERVER_IP>:<PORT>/ws`. Use the server's network IP, not `localhost`, when connecting from another machine.
- Allow the server port (default `8080`) in the server machine's firewall, for the local network only.

_Documented on: 2026-11-13_
