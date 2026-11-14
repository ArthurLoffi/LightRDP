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
- Protecting the host with an access password.
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
You will see a line with `servidor de sinalização ouvindo`. Leave this terminal open.

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
Conectado ao servidor como "HOST"
Aguardando conexões...
```
Tell the person who will connect the ID you chose (`HOST` in this example).

## 3. Connect to the host
On the other computer:
```
go run ./cmd/client -mode join -peer HOST -server ws://<SERVER_IP>:8080/ws
```
You will see:
```
Conectado ao servidor como "<YOUR_ID>"
Pedido enviado para "HOST"
```
On the host, `Pedido de conexão de "<YOUR_ID>"` appears.

When the connection is ready, both computers show:
```
Peer connection state changed: connected

 DataChannel 'chat' aberto! Digite mensagens:
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
erro: peer connection failed
```
This is expected. It may also show `aviso do servidor: destino não encontrado: <ID>` before that, which can be ignored.

After a session ends, the host stops. To accept a new connection, start the host again.

Press `Ctrl+C` in the server terminal to stop the server.

## Things to know
- **One session at a time.** While the host is in a session, other computers that try to connect get `erro: conexão recusada: host ocupado`. See [Host busy](../errors/host_busy.md).
- **No access password yet.** Anyone on your network who knows the host ID can connect to it. Only run the host on networks you trust, and close it when you are done.
- **The server can go away.** Once the two computers are connected, the chat keeps working even if the server stops. The client shows `Servidor de sinalização desconectado; chat P2P continua.`.
- **Each ID can be used once.** Two computers cannot use the same ID at the same time. See [ID already registered](../errors/id_already_registered.md).

## Network
- The server computer must accept connections on the server port (`8080` by default, TCP).
- The host and the client must be able to send UDP traffic to each other.
- Firewalls such as `ufw` or `firewalld` may block this traffic. Only allow it from your local network, never from the internet.

## Something went wrong?
| What you see | Page |
|---|---|
| `uso: client -mode host\|join ...` or `-peer é obrigatório em -mode join` | [Invalid arguments](../errors/invalid_arguments.md) |
| `erro: transport: dial ...` | [Cannot reach the signaling server](../errors/server_unreachable.md) |
| `erro: servidor: id já registrado` | [ID already registered](../errors/id_already_registered.md) |
| `erro: servidor: destino não encontrado: <ID>` | [Host not found](../errors/peer_not_found.md) |
| `erro: conexão recusada: host ocupado` | [Host busy](../errors/host_busy.md) |
| `erro: servidor de sinalização desconectado: ...` | [Signaling server disconnected](../errors/server_disconnected.md) |
| `erro: peer connection failed` | [Peer connection failed](../errors/peer_connection_failed.md) |

All known errors are listed in [Known errors](../errors/README.md).
