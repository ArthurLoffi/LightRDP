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
- Install `openssl` on the server computer, to create its certificate.
- Download the LightRDP project on each computer, and open a terminal in the project folder.
- All computers must be on the same network.
- Check the network rules in [Network](#network).

## 1. Start the server
On the computer that will run the server, replace `<SERVER_IP>` with its IP address on the network (for example `192.168.0.10`).

First create the certificate that encrypts the connection to the server:
```
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 365 \
  -keyout server.key -out server.crt -subj "/CN=LightRDP" \
  -addext "subjectAltName=IP:<SERVER_IP>"
```
Copy `server.crt` to the project folder of the host and of the client. It is public. Never copy `server.key`: it stays on the server computer. See [Security](security/README.md) for details.

Then start the server:
```
go run ./cmd/lightrdp server --bind <SERVER_IP> --cert server.crt --key server.key
```
You will see a line with `signaling server listening` and `tls=true`. Leave this terminal open.

| Option | What it does | Default |
|---|---|---|
| `--bind` | Network address the server listens on. `127.0.0.1` only accepts connections from the same computer. Use the computer's network IP to accept connections from the network, or `0.0.0.0` for every network connection of the computer. | `127.0.0.1` |
| `--port` | Port the server listens on. | `8080` |
| `--cert` | Certificate file. Encrypts connections (`wss://`). Required with `--key` when `--bind` is not `127.0.0.1`, `::1` or `localhost`. | none |
| `--key` | Private key file of the certificate. Required with `--cert`. | none |
| `--debug` | Shows more detailed messages. | `false` |

Example with another port:
```
go run ./cmd/lightrdp server --bind <SERVER_IP> --port 9000 --cert server.crt --key server.key
```

In the steps below, replace `<SERVER_IP>` with the same IP address.

## 2. Start the host
On the computer that will be accessed:
```
go run ./cmd/lightrdp host --id HOST --server wss://<SERVER_IP>:8080/ws --ca server.crt
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
go run ./cmd/lightrdp join HOST --server wss://<SERVER_IP>:8080/ws --ca server.crt
```
You will see:
```
Connected to server as "<YOUR_ID>"
Request sent to "HOST"
Waiting for host confirmation...
```

## 4. Compare the safety code and accept
On the host, this appears:
```
Connection request from "<YOUR_ID>"
```
The two computers then connect, exchange random values for the code, and both show the same safety code:
```
Peer connection state changed: connected
Safety code: 482 913
```
The code is different on every connection. Both computers must run the same LightRDP version (`lightrdp version`): older versions cannot connect to newer ones.
The host is asked:
```
Accept "<YOUR_ID>" with this code? [y/N]
```
The client is asked:
```
Same code on the host? [y/N]
```
Compare the code by phone, in person, or another way that does not go through LightRDP. Each person types `y` and presses Enter only if the code is the same. On the host, also check the ID: only accept who you expect. Any other answer, or just Enter, closes the connection on both computers. If the codes are different, someone may be between the two computers: see [Safety code not confirmed](../errors/safety_code_not_confirmed.md).

Anything typed before the question appears, or in the first second after it, is ignored, and the question is shown again after `Answer ignored: too fast.`. Just answer again. This keeps a late answer meant for an earlier request from accepting someone else.

The host must answer within 60 seconds of the request. After a refusal, the host keeps waiting for new requests.

| What happens | Host shows | Client shows |
|---|---|---|
| Both answer `y` | The chat opens | The chat opens |
| Host answers anything else | `Request from "<YOUR_ID>" refused.` | `error: connection refused: refused by host` |
| Client answers anything else | `error: connection refused: safety code not confirmed` | `error: safety code not confirmed` |
| Host does not answer in 60 seconds | `Request from "<YOUR_ID>" refused: timed out.` | `error: connection refused: timed out waiting for host` |

When both have answered `y`, both computers show:
```
 DataChannel 'chat' open! Type messages:
```
You can now use the [Chat](chat/README.md). Nothing typed before this line is sent.

## Client commands and options
`lightrdp host` waits for someone to connect. `lightrdp join HOST_ID` connects to the host with that ID; the host ID is required.

| Option | What it does | Default |
|---|---|---|
| `--id` | Your ID on the server. If left empty, a random 6-character ID is created, for example `FO3TFX`. | random |
| `--server` | Address of the server. Use `wss://` for a server on another computer. `ws://` is only accepted for the same computer (`localhost`, `127.0.0.1`, `::1`). | `ws://localhost:8080/ws` |
| `--ca` | Certificate file to trust for `wss://`, for example the server's `server.crt`. Needed when the server uses a self-signed certificate. | none |

If the server is on the same computer as the client and was started without `--bind`, you can leave out `--server`. If you changed the server port with `--port`, use the same port in `--server`.

Every command shows its options, defaults and examples with `--help`, for example `go run ./cmd/lightrdp join --help`. `go run ./cmd/lightrdp version` shows the version.

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
- **One session at a time.** While the host is in a session, or is connecting to or deciding on a request, other computers that try to connect get `error: connection refused: host busy`. See [Host busy](../errors/host_busy.md).
- **Only accept who you expect.** Anyone on your network who knows the host ID can send a request. Only answer `y` when you know who is connecting and the safety code is the same. Only run the host on networks you trust, and close it when you are done.
- **The host connects before it is asked.** Anyone who knows the host ID can make the host connect and learn its IP addresses before the host answers. No chat message is exchanged before both people confirm the code.
- **The server can go away.** Once the two computers are connected, the chat keeps working even if the server stops. The client shows `Signaling server disconnected; P2P chat continues.`.
- **Each ID can be used once.** Two computers cannot use the same ID at the same time. See [ID already registered](../errors/id_already_registered.md).

## Network
- The server computer must accept connections on the server port (`8080` by default, TCP).
- The host and the client must be able to send UDP traffic to each other.
- Firewalls such as `ufw` or `firewalld` may block this traffic. Only allow it from your local network, never from the internet.

## Something went wrong?
| What you see | Page |
|---|---|
| `Error: accepts 1 arg(s), received 0`, `Error: unknown command ...` or `Error: unknown flag: ...` | [Invalid arguments](../errors/invalid_arguments.md) |
| `error: transport: insecure server URL ws://<ADDRESS>/ws: use wss://` | [Insecure server address](../errors/insecure_server_url.md) |
| `error: transport: dial wss://... x509: certificate signed by unknown authority` | [Server certificate not trusted](../errors/untrusted_certificate.md) |
| `error: transport: dial wss://... x509: certificate is valid for <NAME>, not <ADDRESS>` | [Certificate does not match the server address](../errors/certificate_name_mismatch.md) |
| `error: --ca: ...` | [Invalid certificate file](../errors/invalid_ca_file.md) |
| `error: transport: dial ...` | [Cannot reach the signaling server](../errors/server_unreachable.md) |
| `error: server: id already registered` | [ID already registered](../errors/id_already_registered.md) |
| `error: server: target not found: <ID>` | [Host not found](../errors/peer_not_found.md) |
| `error: connection refused: host busy` | [Host busy](../errors/host_busy.md) |
| `error: connection refused: refused by host` | [Refused by the host](../errors/refused_by_host.md) |
| `error: connection refused: timed out waiting for host` | [Host did not answer in time](../errors/confirmation_timeout.md) |
| `error: safety code not confirmed` or `error: connection refused: safety code not confirmed` | [Safety code not confirmed](../errors/safety_code_not_confirmed.md) |
| `error: safety code commitment mismatch` or `Request from "<ID>" dropped: safety code commitment mismatch` | [Safety code commitment mismatch](../errors/safety_code_commitment_mismatch.md) |
| `error: safety code handshake failed: ...` or `Request from "<ID>" dropped: safety code handshake failed: ...` | [Safety code exchange failed](../errors/safety_code_handshake_failed.md) |
| `error: signaling server disconnected: ...` | [Signaling server disconnected](../errors/server_disconnected.md) |
| `error: peer connection failed` | [Peer connection failed](../errors/peer_connection_failed.md) |
| Server: `--cert and --key must be used together` | [Certificate or key missing](../errors/tls_flags_incomplete.md) |
| Server: `--cert and --key are required when --bind is not a loopback address` | [Encryption required on the network](../errors/tls_required.md) |
| Server: `server stopped err=... tls: ...` | [Server cannot load the certificate](../errors/server_tls_files.md) |

All known errors are listed in [Known errors](../errors/README.md).
