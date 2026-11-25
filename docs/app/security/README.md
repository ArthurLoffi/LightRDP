# Security
How LightRDP protects the connection between the two computers, and what you need to set up.

There are two protections, and both are needed:
- **Encrypted server connection (`wss://`)**: nobody on the network between a computer and the server can read or change what they send.
- **Safety code**: proves that the connection really ends at the other person's computer, even if the server itself was changed by an attacker.

## Encrypted server connection
When the server accepts connections from the network, it must use a certificate. The server refuses to start on a network address without one.

### 1. Create the certificate
On the server computer, in the project folder, replace `<SERVER_IP>` with its IP address on the network (for example `192.168.0.10`):
```bash
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 365 \
  -keyout server.key -out server.crt -subj "/CN=LightRDP" \
  -addext "subjectAltName=IP:<SERVER_IP>"
```
This creates two files:
- `server.crt`: the certificate. It is public. Copy it to every computer that runs a client.
- `server.key`: the private key. Keep it only on the server computer and never share it.

The certificate is valid for 365 days and only for the IP address written in it. If the server IP changes, create the certificate again and copy the new `server.crt` to the clients.

To use a name instead of an IP, write `subjectAltName=DNS:<NAME>`, and use the same name in `--server`.

### 2. Start the server with the certificate
```bash
go run ./cmd/lightrdp server --bind <SERVER_IP> --cert server.crt --key server.key
```
The server line shows `tls=true`.

### 3. Start the clients with the certificate
On each client computer, with `server.crt` copied to the project folder:
```bash
go run ./cmd/lightrdp host --id HOST --server wss://<SERVER_IP>:8080/ws --ca server.crt
go run ./cmd/lightrdp join HOST --server wss://<SERVER_IP>:8080/ws --ca server.crt
```
`--ca` tells the client to trust this certificate. Without it, the client only trusts certificates from public certificate authorities and refuses the self-signed one. The client always checks the certificate; there is no option to skip the check.

### Same computer only
If the server and the clients all run on the same computer, no certificate is needed: start the server without `--bind` (it listens on `127.0.0.1`) and leave out `--server` in the clients. `ws://` is only accepted for `localhost`, `127.0.0.1` and `::1`.

## Safety code
After the two computers connect, both show the same 6-digit code, for example:
```
Safety code: 482 913
```
The code is calculated from the encryption keys the two computers actually used to connect, and from a random value chosen by each computer. If someone is in the middle, the codes on the two screens are different.

### How the random values are exchanged
After the two computers connect, and before the code is shown, they exchange the random values over the encrypted connection, in this order:
1. The client sends a fingerprint (SHA-256) of its random value, without the value itself.
2. The host sends its random value.
3. The client reveals its random value. The host checks that it matches the fingerprint from step 1.

The client is bound to its value before it sees the host's value, and the host sends its value without knowing the client's value. This stops someone in the middle from trying many encryption keys until the two codes match: whatever they do, the codes match by chance only once in a million connection attempts, and every failed attempt is visible to both people. If the check in step 3 fails, the connection closes with [Safety code commitment mismatch](../../errors/safety_code_commitment_mismatch.md). If a message is missing, repeated or out of order, it closes with [Safety code exchange failed](../../errors/safety_code_handshake_failed.md).

Both computers must run a LightRDP version with this exchange. Older versions cannot connect to newer ones.

1. Compare the code with the other person by phone, in person, or another way that does not go through LightRDP.
2. On the host, answer `y` to `Accept "<ID>" with this code? [y/N]` only if the code is the same and you know who is connecting.
3. On the client, answer `y` to `Same code on the host? [y/N]` only if the code is the same.

The chat opens only after both people answer `y`. Any other answer closes the connection on both computers.

If the codes are different, answer `n`, check the server address and certificate, and try again. If they are different again, do not use that network or that server.

## Things to know
- The host is asked only after the two computers are connected. Anyone who knows the host ID can make the host connect and learn the host's IP addresses before the host answers. No chat message is exchanged before both people confirm.
- The server still sees the IP addresses of both computers.
- A new code is calculated for every connection, even between the same two computers.
