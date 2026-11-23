# Certificate or key missing
**Message:** `--cert and --key must be used together`

## Description
The server did not start because only one of `--cert` and `--key` was given.

## Cause
To encrypt connections, the server needs both the certificate and its private key.

## Possible solution
- Pass both: `go run ./cmd/lightrdp server --bind <SERVER_IP> --cert server.crt --key server.key`. See [Security](../app/security/README.md).

_Documented on: 2026-11-18 (updated 2026-11-21)_
