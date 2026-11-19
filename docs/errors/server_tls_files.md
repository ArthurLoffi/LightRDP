# Server cannot load the certificate
**Message:** `server stopped err=... tls: ...` (or `server stopped err=open <FILE>: ...`)

## Description
The server stopped right after starting because it could not use the certificate or the key.

## Cause
- The certificate or key file does not exist or cannot be read.
- The key does not belong to the certificate (`tls: private key does not match public key`).
- One of the files is not in PEM format.

## Possible solution
- Check the paths given in `-cert` and `-key`.
- Make sure both files were created by the same `openssl` command. If in doubt, create them again. See [Security](../app/security/README.md).

_Documented on: 2026-11-18_
