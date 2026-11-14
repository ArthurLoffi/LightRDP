# Signaling server disconnected
**Message:** `erro: servidor de sinalização desconectado: ...`

## Description
The connection to the signaling server was lost before the two machines finished connecting to each other.

## Cause
- The server was stopped or restarted.
- The network connection to the server was lost.

## Possible solution
- Make sure the server is running, then start the client again.

If the two machines were already connected, the chat keeps working and the client only shows "Servidor de sinalização desconectado; chat P2P continua.".

_Documented on: 2026-11-13_
