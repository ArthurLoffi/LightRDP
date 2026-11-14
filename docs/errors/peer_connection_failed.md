# Peer connection failed
**Message:** `erro: peer connection failed`

## Description
The two machines found each other through the server, but could not open a direct connection, or the connection was lost.

## Cause
- A firewall is blocking UDP traffic between the machines.
- The machines are on different networks that do not allow direct connections.
- The other machine closed or lost its connection for too long.

## Possible solution
- Allow UDP traffic between the two machines in their firewalls, for the local network only.
- Make sure both machines are on the same network.
- Start both clients again.

_Documented on: 2026-11-13_
