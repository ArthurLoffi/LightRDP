# Chat
Send text messages between the host and the client during a session.

## Starting the chat
The chat opens after the two computers connect and both people confirm the safety code with `y`. Both show:
```
 DataChannel 'chat' open! Type messages:
```
Nothing typed before this line is sent, and nothing is shown from the other computer before it. To connect the computers and compare the code, see [Getting Started](../getting_started.md) and [Security](../security/README.md).

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
