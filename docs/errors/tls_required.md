# Encryption required on the network
**Message:** `-cert and -key are required when -bind is not a loopback address`

## Description
The server did not start because it would accept connections from the network without encryption.

## Cause
`-bind` is a network address (for example `192.168.0.10`, `0.0.0.0` or `::`), and `-cert` and `-key` were not given. Without encryption, anyone on the network could read or change the connection data.

## Possible solution
- Create a certificate and start the server with `-cert` and `-key`. See [Security](../app/security/README.md).
- Or, if everything runs on the same computer, use `-bind 127.0.0.1` (the default).

_Documented on: 2026-11-18_
