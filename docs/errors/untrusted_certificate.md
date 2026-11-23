# Server certificate not trusted
**Message:** `error: transport: dial wss://<ADDRESS>/ws: ... x509: certificate signed by unknown authority`

## Description
The client does not trust the certificate the server presented, so it did not connect.

## Cause
The server uses a self-signed certificate, or one from a private certificate authority, and the client was started without `--ca`, or with a different certificate. It can also mean that something on the network is pretending to be the server.

## Possible solution
- Copy the server's `server.crt` file to this computer and pass it with `--ca server.crt`. See [Security](../app/security/README.md).
- Make sure the file is the one the server is using now. If the certificate was created again, copy the new one.
- If you already pass the right file and still see this error, do not use that network.

_Documented on: 2026-11-18 (updated 2026-11-21)_
