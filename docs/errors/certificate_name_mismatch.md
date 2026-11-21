# Certificate does not match the server address
**Messages:**
- `error: transport: dial wss://<ADDRESS>/ws: ... x509: certificate is valid for <NAME>, not <ADDRESS>`
- `error: transport: dial wss://<ADDRESS>/ws: ... x509: certificate is not valid for any names, but wanted to match <ADDRESS>` (the certificate only has IP addresses and `--server` uses a name such as `localhost`)

## Description
The server certificate was made for another address, so the client did not connect.

## Cause
The address used in `--server` is not in the certificate's `subjectAltName`. For example, the certificate was made for `192.168.0.10` and the client connects to `192.168.0.20`, or to a name instead of the IP.

## Possible solution
- Use in `--server` the same IP or name written in the certificate.
- Or create the certificate again with the right address in `subjectAltName`, and copy the new `server.crt` to each client. See [Security](../app/security/README.md).

_Documented on: 2026-11-18 (updated 2026-11-21)_
