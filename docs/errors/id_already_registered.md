# ID already registered
**Message:** `erro: servidor: id já registrado`

## Description
The client could not join the signaling server because another machine is already using the same ID.

## Cause
Two clients were started with the same `-id`, or a previous session with that ID is still running.

## Possible solution
- Choose a different ID with `-id`, or leave it out to get a random one.
- Close the other session that is using this ID and try again.

_Documented on: 2026-11-13_
