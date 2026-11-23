# Invalid certificate file
**Message:** `error: --ca: <CAUSE>`

## Description
The client could not use the file given in `--ca`.

## Cause
- The file does not exist or cannot be read (for example `open server.crt: no such file or directory`).
- The file has no valid PEM certificate (`no valid PEM certificate in <FILE>`). This happens, for example, when the key file (`server.key`) is passed instead of the certificate.

## Possible solution
- Check the path to the file.
- Pass the certificate file (`server.crt`, which starts with `-----BEGIN CERTIFICATE-----`), never the key.

_Documented on: 2026-11-18 (updated 2026-11-21)_
