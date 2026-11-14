# Host not found
**Message:** `erro: servidor: destino não encontrado: <ID>`

## Description
The machine you are trying to reach is not connected to the signaling server.

## Cause
- The host ID was typed wrong.
- The host is not running, or it is connected to a different server.

## Possible solution
- Check the ID passed to `-peer`. IDs are case-sensitive.
- Make sure the host is running and shows "Aguardando conexões...".
- Make sure both machines use the same `-server` address.

_Documented on: 2026-11-13_
