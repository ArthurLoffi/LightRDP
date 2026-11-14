# Chat
Send text messages between the host and the client during a session.

## Starting the chat
The chat opens on its own when the two computers connect. Both show:
```
 DataChannel 'chat' aberto! Digite mensagens:
```
To connect the computers, see [Getting Started](../getting_started.md).

## Sending a message
Type the message and press `Enter`. Empty lines are not sent.

## Receiving a message
Messages from the other computer appear with `< ` in front:
```
< hello, can you see this?
```

## Things to know
- Messages go straight from one computer to the other. They do not pass through the server.
- The chat keeps working if the server stops after the computers are connected.
- Special control characters in received messages are removed before they are shown, so the other computer cannot change how your terminal looks or behaves. Normal text and tabs are kept.
- Messages are not saved. When the session ends, they are gone.
