# Getting Started
How to connect two computers with LightRDP and start a session.

A session needs three parts:
- **Server**: helps the two computers find each other. It can run on either computer or on a third one.
- **Host**: the computer that will be accessed.
- **Client**: the computer that connects to the host.

## What you can do today
- Connect two computers on the same network.
- Send text messages between them. See [Chat](chat/README.md).

### Not available yet
- Seeing and controlling the other computer's screen.
- Sharing resources as the host.
- Protecting the host with an access password. For now, the host confirms each connection by hand.
- Sharing copied text between the computers.
- Session history.

## Before you start
- Install [Go](https://go.dev/dl/) 1.26.9 or newer on each computer.
- Download the LightRDP project on each computer, and open a terminal in the project folder.
- All computers must be on the same network.
- Check the network rules in [Network](#network).

## 1. Start the server
On the computer that will run the server:
```
go run ./cmd/server
```
You will see a line with `signaling server listening`. Leave this terminal open.

| Option | What it does | Default |
|---|---|---|
| `-addr` | Address and port the server listens on. `:8080` means port 8080 on every network connection of the computer. | `:8080` |
| `-debug` | Shows more detailed messages. | `false` |

Example with another port:
```
go run ./cmd/server -addr :9000
```

You will need the server computer's IP address on the network (for example `192.168.0.10`). In the steps below, replace `<SERVER_IP>` with it.

## 2. Start the host
On the computer that will be accessed:
```
go run ./cmd/client -mode host -id HOST -server ws://<SERVER_IP>:8080/ws
```
You will see:
```
Connected to server as "HOST"
Waiting for connections...
```
Tell the person who will connect the ID you chose (`HOST` in this example).

## 3. Connect to the host
On the other computer:
```
go run ./cmd/client -mode join -peer HOST -server ws://<SERVER_IP>:8080/ws
```
You will see:
```
Connected to server as "<YOUR_ID>"
Request sent to "HOST"
Waiting for host confirmation...
```

## 4. Accept the connection on the host
On the host, this appears:
```
Connection request from "<YOUR_ID>"
Accept? [y/N]
```
Type `y` and press Enter to accept. Any other answer, or just Enter, refuses the request. Nothing is exchanged between the two computers before the host accepts.

Read the ID before answering. Anything typed before the question appears, or in the first second after it, is ignored, and the host shows `Answer ignored: too fast. Accept? [y/N]`. Just answer again. This keeps a late answer meant for an earlier request from accepting someone else.

If nobody answers within 60 seconds, the request is refused. After a refusal, the host keeps waiting for new requests.

| What happens | Host shows | Client shows |
|---|---|---|
| Host answers `y` | The connection starts | The connection starts |
| Host answers anything else | `Request from "<YOUR_ID>" refused.` | `error: connection refused: refused by host` |
| No answer in 60 seconds | `Request from "<YOUR_ID>" refused: timed out.` | `error: connection refused: timed out waiting for host` |

When the connection is ready, both computers show:
```
Peer connection state changed: connected

 DataChannel 'chat' open! Type messages:
```
You can now use the [Chat](chat/README.md).

## Client options
| Option | What it does | Default |
|---|---|---|
| `-mode` | `host` waits for someone to connect. `join` connects to a host. Required. | none |
| `-id` | Your ID on the server. If left empty, a random 6-character ID is created, for example `FO3TFX`. | random |
| `-peer` | ID of the host to connect to. Required with `-mode join`. | none |
| `-server` | Address of the server. | `ws://localhost:8080/ws` |

If the server is on the same computer as the client, you can leave out `-server`. If you changed the server port with `-addr`, use the same port in `-server`.

## Ending a session
Press `Ctrl+C` in the terminal of the host or the client.

The other computer notices after about 30 seconds. It shows:
```
Peer connection state changed: disconnected
Peer connection state changed: failed
error: peer connection failed
```
This is expected. It may also show `server notice: target not found: <ID>` before that, which can be ignored.

After a session ends, the host stops. To accept a new connection, start the host again.

Press `Ctrl+C` in the server terminal to stop the server.

## Things to know
- **One session at a time.** While the host is in a session, or is deciding on a request, other computers that try to connect get `error: connection refused: host busy`. See [Host busy](../errors/host_busy.md).
- **Only accept who you expect.** Anyone on your network who knows the host ID can send a request. Only answer `y` when you know who is connecting. Only run the host on networks you trust, and close it when you are done.
- **The server can go away.** Once the two computers are connected, the chat keeps working even if the server stops. The client shows `Signaling server disconnected; P2P chat continues.`.
- **Each ID can be used once.** Two computers cannot use the same ID at the same time. See [ID already registered](../errors/id_already_registered.md).

## Network
- The server computer must accept connections on the server port (`8080` by default, TCP).
- The host and the client must be able to send UDP traffic to each other.
- Firewalls such as `ufw` or `firewalld` may block this traffic. Only allow it from your local network, never from the internet.

## Something went wrong?
| What you see | Page |
|---|---|
| `usage: client -mode host\|join ...` or `-peer is required with -mode join` | [Invalid arguments](../errors/invalid_arguments.md) |
| `error: transport: dial ...` | [Cannot reach the signaling server](../errors/server_unreachable.md) |
| `error: server: id already registered` | [ID already registered](../errors/id_already_registered.md) |
| `error: server: target not found: <ID>` | [Host not found](../errors/peer_not_found.md) |
| `error: connection refused: host busy` | [Host busy](../errors/host_busy.md) |
| `error: connection refused: refused by host` | [Refused by the host](../errors/refused_by_host.md) |
| `error: connection refused: timed out waiting for host` | [Host did not answer in time](../errors/confirmation_timeout.md) |
| `error: signaling server disconnected: ...` | [Signaling server disconnected](../errors/server_disconnected.md) |
| `error: peer connection failed` | [Peer connection failed](../errors/peer_connection_failed.md) |

All known errors are listed in [Known errors](../errors/README.md).
